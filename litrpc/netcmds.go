package litrpc

import (
	"fmt"
	"strconv"

	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/qln"
)

// ------------------------- listen

type ListenArgs struct {
	Port string
}

func (r *LitRPC) Listen(args ListenArgs, reply *StatusReply) error {
	if args.Port == "" {
		args.Port = ":2448"
	}
	adr, err := r.Node.TCPListener(args.Port)
	if err != nil {
		return err
	}
	// todo: say what port and what pubkey in status message
	reply.Status = fmt.Sprintf("listening on %s with key %s",
		args.Port, adr.String())
	return nil
}

// ------------------------- connect
type ConnectArgs struct {
	LNAddr string
}

func (r *LitRPC) Connect(args ConnectArgs, reply *StatusReply) error {

	// first, see if the peer to connect to is referenced by peer index.
	var connectAdr *lndc.LNAdr
	var peerIdx uint32

	// check if a peer number was supplied instead of a pubkeyhash
	peerIdxint, err := strconv.Atoi(args.LNAddr)
	// number will mean no error
	if err == nil {
		pubArr, host := r.Node.GetPubHostFromPeerIdx(uint32(peerIdxint))
		adrString := fmt.Sprintf("%x", pubArr)
		if host != "" {
			adrString += "@" + host
		}
		fmt.Printf("try string %s\n", adrString)
		// pretty ugly to build the string here...
		connectAdr, err = lndc.LnAddrFromString(adrString)
		if err != nil {
			return err
		}
		// get peer from address book
	} else {
		connectAdr, err = lndc.LnAddrFromString(args.LNAddr)
		if err != nil {
			return err
		}
	}

	// get my private ID key
	idPriv := r.Node.IdKey()

	// Assign remote connection
	newConn := new(lndc.LNDConn)

	err = newConn.Dial(idPriv,
		connectAdr.NetAddr.String(), connectAdr.Base58Adr.ScriptAddress())
	if err != nil {
		return err
	}
	// if connect is successful, either query for already existing peer index, or
	// if the peer is new, make an new index, and save the hostname&port

	// figure out peer index, or assign new one for new peer.  Since
	// we're connecting out, also specify the hostname&port
	peerIdx, err = r.Node.GetPeerIdx(newConn.RemotePub, newConn.RemoteAddr().String())
	if err != nil {
		return err
	}

	r.Node.RemoteMtx.Lock()
	r.Node.RemoteCons[peerIdx] = newConn
	r.Node.RemoteMtx.Unlock()

	// each connection to a peer gets its own LNDCReader
	go r.Node.LNDCReader(newConn, peerIdx)

	reply.Status = fmt.Sprintf("connected to peer %d", peerIdx)
	return nil
}

// ------------------------- ShowConnections

type ListConnectionsReply struct {
	Connections []qln.PeerInfo
}
type ConInfo struct {
	PeerNumber uint32
	RemoteHost string
}

func (r *LitRPC) ListConnections(args NoArgs, reply *ListConnectionsReply) error {
	reply.Connections = r.Node.GetConnectedPeerList()

	return nil
}

// ------- receive chat
func (r *LitRPC) GetMessages(args NoArgs, reply *StatusReply) error {
	reply.Status = <-r.Node.UserMessageBox
	return nil
}

type SayArgs struct {
	Peer    uint32
	Message string
}

func (r *LitRPC) Say(args SayArgs, reply *StatusReply) error {
	return r.Node.SendChat(args.Peer, args.Message)
}

func (r *LitRPC) Stop(args NoArgs, reply *StatusReply) error {
	reply.Status = "Stopping lit node"
	r.OffButton <- true
	return nil
}
