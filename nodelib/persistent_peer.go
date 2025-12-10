package nodelib

type PersistentPeer struct {
	NodeId      string
	NodeAddress string
	P2PPort     uint16
}

type PersistentPeers []PersistentPeer

func NewPersistentPeers(persistentPeers string) PersistentPeers {

	//nodeid@nodeaddress:p2p_port,nodeid2@nodeaddress2:p2p_port2

	return nil
}

func (p PersistentPeers) GetNodeHostById(nodeId string) string {
	return ""
}
