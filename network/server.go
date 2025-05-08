package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/deauthe/local_blockchain_go/api"
	"github.com/deauthe/local_blockchain_go/core"
	"github.com/deauthe/local_blockchain_go/crypto"
	"github.com/deauthe/local_blockchain_go/types"
	"github.com/go-kit/log"
)

var defaultBlockTime = 5 * time.Second

type ServerOpts struct {
	APIListenAddr string
	SeedNodes     []string
	ListenAddr    string
	TCPTransport  *TCPTransport
	ID            string
	Logger        log.Logger
	RPCDecodeFunc RPCDecodeFunc
	RPCProcessor  RPCProcessor
	BlockTime     time.Duration
	PrivateKey    *crypto.PrivateKey
}

type Server struct {
	TCPTransport *TCPTransport
	peerCh       chan *TCPPeer

	mu      sync.RWMutex
	peerMap map[net.Addr]*TCPPeer

	ServerOpts
	mempool     *TxPool
	chain       *core.Blockchain
	isValidator bool
	rpcCh       chan RPC
	quitCh      chan struct{}
	txChan      chan *core.Transaction
}

func NewServer(opts ServerOpts) (*Server, error) {
	if opts.BlockTime == time.Duration(0) {
		opts.BlockTime = defaultBlockTime
	}
	if opts.RPCDecodeFunc == nil {
		opts.RPCDecodeFunc = DefaultRPCDecodeFunc
	}
	if opts.Logger == nil {
		opts.Logger = log.NewLogfmtLogger(os.Stderr)
		opts.Logger = log.With(opts.Logger, "addr", opts.ID)
	}

	chain, err := core.NewBlockchain(opts.Logger, genesisBlock())
	if err != nil {
		return nil, err
	}

	// Channel being used to communicate between the JSON RPC server
	// and the node that will process this message.
	txChan := make(chan *core.Transaction)

	// Only boot up the API server if the config has a valid port number.
	if len(opts.APIListenAddr) > 0 {
		apiServerCfg := api.ServerConfig{
			Logger:     opts.Logger,
			ListenAddr: opts.APIListenAddr,
		}
		apiServer := api.NewServer(apiServerCfg, chain, txChan)
		go apiServer.Start()

		opts.Logger.Log("msg", "JSON API server running", "port", opts.APIListenAddr)
	}

	peerCh := make(chan *TCPPeer)
	tr := NewTCPTransport(opts.ListenAddr, peerCh)

	s := &Server{
		TCPTransport: tr,
		peerCh:       peerCh,
		peerMap:      make(map[net.Addr]*TCPPeer),
		ServerOpts:   opts,
		chain:        chain,
		mempool:      NewTxPool(1000),
		isValidator:  opts.PrivateKey != nil,
		rpcCh:        make(chan RPC),
		quitCh:       make(chan struct{}, 1),
		txChan:       txChan,
	}

	s.TCPTransport.peerCh = peerCh

	// If we dont got any processor from the server options, we going to use
	// the server as default.
	if s.RPCProcessor == nil {
		s.RPCProcessor = s
	}

	if s.isValidator {
		go s.validatorLoop()
	}

	return s, nil
}

func (s *Server) bootstrapNetwork() {
	for _, addr := range s.SeedNodes {
		fmt.Println("trying to connect to ", addr)

		go func(addr string) {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				fmt.Printf("could not connect to %+v\n", conn)
				return
			}

			s.peerCh <- &TCPPeer{
				conn: conn,
			}
		}(addr)
	}
}

func (s *Server) Start() {
	s.TCPTransport.Start()

	time.Sleep(time.Second * 1)

	s.bootstrapNetwork()

	s.Logger.Log("msg", "accepting TCP connection on", "addr", s.ListenAddr, "id", s.ID)

free:
	for {
		select {
		case peer := <-s.peerCh:
			s.mu.Lock()
			// Check if we already have this peer
			if existingPeer, exists := s.peerMap[peer.conn.RemoteAddr()]; exists {
				existingPeer.Close()
			}
			s.peerMap[peer.conn.RemoteAddr()] = peer
			s.mu.Unlock()

			go peer.readLoop(s.rpcCh)

			if err := s.sendGetStatusMessage(peer); err != nil {
				s.Logger.Log("err", err)
				continue
			}

			s.Logger.Log("msg", "peer added to the server", "outgoing", peer.Outgoing, "addr", peer.conn.RemoteAddr())

		case tx := <-s.txChan:
			fmt.Println("new transaction message: ", tx.TxHash)
			if err := s.processTransaction(tx); err != nil {
				s.Logger.Log("process TX error", err)
			}

		case rpc := <-s.rpcCh:
			msg, err := s.RPCDecodeFunc(rpc)
			if err != nil {
				s.Logger.Log("RPC error", err)
				continue
			}

			if err := s.RPCProcessor.ProcessMessage(msg); err != nil {
				if err != core.ErrBlockKnown {
					s.Logger.Log("error", err)
				}
			}

		case <-s.quitCh:
			break free
		}
	}

	// Cleanup all peers on shutdown
	s.mu.Lock()
	for _, peer := range s.peerMap {
		peer.Close()
	}
	s.mu.Unlock()

	s.Logger.Log("msg", "Server is shutting down")
}

func (s *Server) validatorLoop() {
	s.Logger.Log("msg", "ticker initialized", "blockTime", s.BlockTime)
	ticker := time.NewTicker(s.BlockTime)
	defer ticker.Stop()

	s.Logger.Log(
		"msg", "Starting validator loop",
		"blockTime", s.BlockTime,
		"isValidator", s.isValidator,
	)

	for {
		select {
		case <-ticker.C:
			s.Logger.Log(
				"msg", "validator tick",
				"current_height", s.chain.Height(),
				"mempool_size", s.mempool.PendingCount(),
			)

			if err := s.createNewBlock(); err != nil {
				s.Logger.Log(
					"msg", "failed to create new block",
					"error", err,
					"current_height", s.chain.Height(),
				)
				continue
			}
		case <-s.quitCh:
			s.Logger.Log("msg", "stopping validator loop")
			return
		}
	}
}

func (s *Server) ProcessMessage(msg *DecodedMessage) error {
	switch t := msg.Data.(type) {
	case *core.Transaction:
		return s.processTransaction(t)
	case *core.Block:
		return s.processBlock(t)
	case *GetStatusMessage:
		return s.processGetStatusMessage(msg.From, t)
	case *StatusMessage:
		return s.processStatusMessage(msg.From, t)
	case *GetBlocksMessage:
		return s.processGetBlocksMessage(msg.From, t)
	case *BlocksMessage:
		return s.processBlocksMessage(msg.From, t)
	}

	return nil
}

func (s *Server) processGetBlocksMessage(from net.Addr, data *GetBlocksMessage) error {
	s.Logger.Log(
		"msg", "received getBlocks message",
		"from", from,
		"from_height", data.From,
		"to_height", data.To,
	)

	var blocks []*core.Block
	ourHeight := s.chain.Height()

	// If To is 0, send all blocks from From to our current height
	endHeight := data.To
	if endHeight == 0 {
		endHeight = ourHeight
	}

	// Ensure we don't request blocks beyond our height
	if endHeight > ourHeight {
		endHeight = ourHeight
	}

	// Requested height is too high
	if data.From > ourHeight {
		s.Logger.Log(
			"msg", "requested height too high",
			"requested", data.From,
			"our_height", ourHeight,
		)
		return nil
	}

	// Get blocks in the requested range
	for height := data.From; height <= endHeight; height++ {
		block, err := s.chain.GetBlock(height)
		if err != nil {
			s.Logger.Log(
				"msg", "failed to get block",
				"height", height,
				"error", err,
			)
			continue
		}
		blocks = append(blocks, block)
	}

	if len(blocks) == 0 {
		s.Logger.Log(
			"msg", "no blocks to send",
			"from", from,
			"from_height", data.From,
			"to_height", endHeight,
		)
		return nil
	}

	s.Logger.Log(
		"msg", "sending blocks",
		"from", from,
		"num_blocks", len(blocks),
		"from_height", blocks[0].Height,
		"to_height", blocks[len(blocks)-1].Height,
	)

	blocksMsg := &BlocksMessage{
		Blocks: blocks,
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(blocksMsg); err != nil {
		return fmt.Errorf("failed to encode blocks message: %w", err)
	}

	s.mu.RLock()
	peer, ok := s.peerMap[from]
	if !ok {
		s.mu.RUnlock()
		return fmt.Errorf("peer %s not known", from)
	}

	msg := NewMessage(MessageTypeBlocks, buf.Bytes())
	err := peer.Send(msg.Bytes())
	s.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to send blocks message: %w", err)
	}

	return nil
}

func (s *Server) sendGetStatusMessage(peer *TCPPeer) error {
	var (
		getStatusMsg = new(GetStatusMessage)
		buf          = new(bytes.Buffer)
	)
	if err := gob.NewEncoder(buf).Encode(getStatusMsg); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeGetStatus, buf.Bytes())
	return peer.Send(msg.Bytes())
}

func (s *Server) broadcast(payload []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var disconnectedPeers []net.Addr

	for netAddr, peer := range s.peerMap {
		if err := peer.Send(payload); err != nil {
			s.Logger.Log("msg", "failed to send to peer", "addr", netAddr, "err", err)
			disconnectedPeers = append(disconnectedPeers, netAddr)
		}
	}

	// Clean up disconnected peers
	if len(disconnectedPeers) > 0 {
		s.mu.RUnlock()
		s.mu.Lock()
		for _, addr := range disconnectedPeers {
			if peer, ok := s.peerMap[addr]; ok {
				peer.Close()
				delete(s.peerMap, addr)
				s.Logger.Log("msg", "removed disconnected peer", "addr", addr)
			}
		}
		s.mu.Unlock()
		s.mu.RLock()
	}

	return nil
}

func (s *Server) processBlocksMessage(from net.Addr, data *BlocksMessage) error {
	s.Logger.Log(
		"msg", "processing blocks message",
		"from", from,
		"num_blocks", len(data.Blocks),
	)

	if len(data.Blocks) == 0 {
		s.Logger.Log("msg", "received empty blocks message", "from", from)
		return nil
	}

	// Process blocks in order
	for _, block := range data.Blocks {
		s.Logger.Log(
			"msg", "processing block from message",
			"height", block.Height,
			"hash", block.Hash(core.BlockHasher{}),
			"transactions", len(block.Transactions),
		)

		// First verify the block
		if err := block.Verify(); err != nil {
			s.Logger.Log(
				"msg", "block verification failed",
				"height", block.Height,
				"hash", block.Hash(core.BlockHasher{}),
				"error", err,
			)
			continue
		}

		// Try to add the block to our chain
		if err := s.chain.AddBlock(block); err != nil {
			if err == core.ErrBlockKnown {
				s.Logger.Log(
					"msg", "block already known",
					"height", block.Height,
					"hash", block.Hash(core.BlockHasher{}),
				)
				continue
			}
			s.Logger.Log(
				"msg", "failed to add block",
				"height", block.Height,
				"hash", block.Hash(core.BlockHasher{}),
				"error", err,
			)
			continue
		}

		// Remove any transactions in the block from our mempool
		for _, tx := range block.Transactions {
			s.mempool.Remove(tx.Hash(core.TxHasher{}))
		}

		s.Logger.Log(
			"msg", "successfully processed block from message",
			"height", block.Height,
			"hash", block.Hash(core.BlockHasher{}),
			"transactions", len(block.Transactions),
		)

		// Broadcast the block to other peers
		go s.broadcastBlock(block)
	}

	return nil
}

func (s *Server) processStatusMessage(from net.Addr, data *StatusMessage) error {
	s.Logger.Log(
		"msg", "received status message",
		"from", from,
		"height", data.CurrentHeight,
		"our_height", s.chain.Height(),
	)

	// If our height is lower, request blocks
	if s.chain.Height() < data.CurrentHeight {
		s.Logger.Log(
			"msg", "our height is lower, requesting blocks",
			"our_height", s.chain.Height(),
			"peer_height", data.CurrentHeight,
			"from", from,
		)

		// Request blocks starting from our current height + 1
		msg := &GetBlocksMessage{
			From: s.chain.Height() + 1,
			To:   data.CurrentHeight,
		}

		if err := s.sendGetBlocksMessage(from, msg); err != nil {
			s.Logger.Log(
				"msg", "failed to send get blocks message",
				"error", err,
				"from", from,
			)
			return fmt.Errorf("failed to send get blocks message: %w", err)
		}
	}

	return nil
}

func (s *Server) sendGetBlocksMessage(to net.Addr, msg *GetBlocksMessage) error {
	s.Logger.Log(
		"msg", "sending get blocks message",
		"to", to,
		"from_height", msg.From,
		"to_height", msg.To,
	)

	// Encode the message
	encodedMsg, err := msg.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode get blocks message: %w", err)
	}

	// Create RPC message
	rpcMsg := NewMessage(MessageTypeGetBlocks, encodedMsg)

	// Find the peer
	peer, ok := s.peerMap[to]
	if !ok {
		return fmt.Errorf("peer not found: %s", to)
	}

	// Send the message
	if err := peer.Send(rpcMsg.Bytes()); err != nil {
		return fmt.Errorf("failed to send message to peer: %w", err)
	}

	return nil
}

func (s *Server) broadcastBlock(b *core.Block) error {
	s.Logger.Log(
		"msg", "broadcasting block",
		"height", b.Height,
		"hash", b.Hash(core.BlockHasher{}),
		"transactions", len(b.Transactions),
	)

	buf := &bytes.Buffer{}
	if err := b.Encode(core.NewGobBlockEncoder(buf)); err != nil {
		return fmt.Errorf("failed to encode block: %v", err)
	}

	msg := NewMessage(MessageTypeBlock, buf.Bytes())
	payload := msg.Bytes()

	s.mu.RLock()
	defer s.mu.RUnlock()

	var failedPeers []net.Addr
	for addr, peer := range s.peerMap {
		if err := peer.Send(payload); err != nil {
			s.Logger.Log(
				"msg", "failed to send block to peer",
				"addr", addr,
				"err", err,
			)
			failedPeers = append(failedPeers, addr)
			continue
		}
		s.Logger.Log(
			"msg", "sent block to peer",
			"addr", addr,
			"height", b.Height,
			"hash", b.Hash(core.BlockHasher{}),
		)
	}

	// Clean up disconnected peers
	if len(failedPeers) > 0 {
		s.mu.RUnlock()
		s.mu.Lock()
		for _, addr := range failedPeers {
			if peer, ok := s.peerMap[addr]; ok {
				peer.Close()
				delete(s.peerMap, addr)
				s.Logger.Log("msg", "removed disconnected peer", "addr", addr)
			}
		}
		s.mu.Unlock()
		s.mu.RLock()
	}

	return nil
}

func (s *Server) broadcastTx(tx *core.Transaction) error {
	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeTx, buf.Bytes())

	return s.broadcast(msg.Bytes())
}

func (s *Server) createNewBlock() error {
	currentHeader, err := s.chain.GetHeader(s.chain.Height())
	fmt.Print("creating a block")
	defer func() {
		if r := recover(); r != nil {
			s.Logger.Log("msg", "panic in validator loop", "error", r)
		}
	}()

	if err != nil {
		s.Logger.Log(
			"msg", "failed to get current header",
			"error", err,
			"height", s.chain.Height(),
		)
		return fmt.Errorf("failed to get current header: %v", err)
	}

	// Get pending transactions from mempool
	txx := s.mempool.Pending()
	s.Logger.Log(
		"msg", "creating new block",
		"current_height", currentHeader.Height,
		"next_height", currentHeader.Height+1,
		"transactions", len(txx),
		"mempool_size", s.mempool.PendingCount(),
	)

	// Create new block
	block, err := core.NewBlockFromPrevHeader(currentHeader, txx)
	if err != nil {
		s.Logger.Log(
			"msg", "failed to create new block",
			"error", err,
			"current_height", currentHeader.Height,
		)
		return fmt.Errorf("failed to create new block: %v", err)
	}

	s.Logger.Log(
		"msg", "created new block",
		"height", block.Height,
		"prev_hash", block.PrevBlockHash,
		"data_hash", block.DataHash,
		"transactions", len(block.Transactions),
	)

	// Double check the block height
	if block.Height != currentHeader.Height+1 {
		s.Logger.Log(
			"msg", "invalid block height",
			"expected", currentHeader.Height+1,
			"got", block.Height,
		)
		return fmt.Errorf("invalid block height: expected %d, got %d", currentHeader.Height+1, block.Height)
	}

	// Sign the block with our validator key
	if err := block.Sign(*s.PrivateKey); err != nil {
		s.Logger.Log(
			"msg", "failed to sign block",
			"height", block.Height,
			"error", err,
		)
		return fmt.Errorf("failed to sign block: %v", err)
	}

	s.Logger.Log(
		"msg", "signed block",
		"height", block.Height,
		"validator", block.Validator,
	)

	// Add block to our chain
	if err := s.chain.AddBlock(block); err != nil {
		if err == core.ErrBlockKnown {
			s.Logger.Log(
				"msg", "block already known",
				"height", block.Height,
				"hash", block.Hash(core.BlockHasher{}),
			)
			return nil
		}
		s.Logger.Log(
			"msg", "failed to add block",
			"height", block.Height,
			"hash", block.Hash(core.BlockHasher{}),
			"error", err,
			"current_height", s.chain.Height(),
		)
		return fmt.Errorf("failed to add block: %v", err)
	}

	// Clear pending transactions that were included in the block
	s.mempool.ClearPending()

	// Broadcast the new block to peers synchronously
	if err := s.broadcastBlock(block); err != nil {
		s.Logger.Log(
			"msg", "failed to broadcast block",
			"height", block.Height,
			"hash", block.Hash(core.BlockHasher{}),
			"error", err,
		)
	}

	s.Logger.Log(
		"msg", "created and added new block",
		"height", block.Height,
		"hash", block.Hash(core.BlockHasher{}),
		"transactions", len(block.Transactions),
		"current_height", s.chain.Height(),
	)

	return nil
}

func genesisBlock() *core.Block {
	header := &core.Header{
		Version:   1,
		DataHash:  types.Hash{},
		Height:    0,
		Timestamp: 000000,
	}

	b, _ := core.NewBlock(header, nil)

	coinbase := crypto.PublicKey{}
	tx := core.NewTransaction(nil)
	tx.From = coinbase
	tx.To = coinbase
	tx.Value = 10_000_000
	b.Transactions = append(b.Transactions, tx)

	privKey := crypto.GeneratePrivateKey()
	if err := b.Sign(privKey); err != nil {
		panic(err)
	}

	return b
}

func (s *Server) processBlock(b *core.Block) error {
	s.Logger.Log(
		"msg", "processing block",
		"height", b.Height,
		"current_height", s.chain.Height(),
		"hash", b.Hash(core.BlockHasher{}),
		"transactions", len(b.Transactions),
	)

	// First verify the block
	if err := b.Verify(); err != nil {
		s.Logger.Log(
			"msg", "block verification failed",
			"height", b.Height,
			"hash", b.Hash(core.BlockHasher{}),
			"error", err,
		)
		return fmt.Errorf("block verification failed: %v", err)
	}

	// Try to add the block to our chain
	if err := s.chain.AddBlock(b); err != nil {
		if err == core.ErrBlockKnown {
			s.Logger.Log(
				"msg", "block already known",
				"height", b.Height,
				"hash", b.Hash(core.BlockHasher{}),
			)
			return nil
		}
		s.Logger.Log(
			"msg", "failed to add block",
			"height", b.Height,
			"hash", b.Hash(core.BlockHasher{}),
			"error", err,
		)
		return fmt.Errorf("failed to add block: %v", err)
	}

	// Remove any transactions in the block from our mempool
	for _, tx := range b.Transactions {
		s.mempool.Remove(tx.Hash(core.TxHasher{}))
	}

	// Broadcast the block to other peers synchronously
	if err := s.broadcastBlock(b); err != nil {
		s.Logger.Log(
			"msg", "failed to broadcast block",
			"height", b.Height,
			"hash", b.Hash(core.BlockHasher{}),
			"error", err,
		)
	}

	s.Logger.Log(
		"msg", "successfully processed block",
		"height", b.Height,
		"current_height", s.chain.Height(),
		"hash", b.Hash(core.BlockHasher{}),
		"transactions", len(b.Transactions),
	)

	return nil
}

func (s *Server) processTransaction(tx *core.Transaction) error {
	hash := tx.TxHash

	if s.mempool.Contains(hash) {
		return nil
	}

	if err := tx.Verify(); err != nil {
		return err
	}

	go s.broadcastTx(tx)

	s.mempool.Add(tx)

	return nil
}

func (s *Server) processGetStatusMessage(from net.Addr, data *GetStatusMessage) error {
	s.Logger.Log("msg", "received getStatus message", "from", from)

	statusMessage := &StatusMessage{
		CurrentHeight: s.chain.Height(),
		ID:            s.ID,
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(statusMessage); err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	peer, ok := s.peerMap[from]
	if !ok {
		return fmt.Errorf("peer %s not known", from)
	}

	msg := NewMessage(MessageTypeStatus, buf.Bytes())

	return peer.Send(msg.Bytes())
}

func (msg *GetBlocksMessage) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to encode GetBlocksMessage: %w", err)
	}
	return buf.Bytes(), nil
}
