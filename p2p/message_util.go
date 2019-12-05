package p2p

var msgTypesDict = make(map[uint64]string)

func init() {
	msgTypesDict[0x00] = "handshakeMsg"
	msgTypesDict[0x01] = "discMsg"
	msgTypesDict[0x02] = "pingMsg"
	msgTypesDict[0x03] = "pongMsg"

	baseProtocolLength := uint64(16)
	// Protocol messages
	msgTypesDict[baseProtocolLength+0x00] = "StatusMsg"
	msgTypesDict[baseProtocolLength+0x01] = "NewBlockHashesMsg"
	msgTypesDict[baseProtocolLength+0x02] = "TxMsg"
	msgTypesDict[baseProtocolLength+0x03] = "GetBlocksMsg"
	msgTypesDict[baseProtocolLength+0x04] = "BlocksMsg"
	msgTypesDict[baseProtocolLength+0x05] = "GetNodeDataMsg"
	msgTypesDict[baseProtocolLength+0x06] = "NodeDataMsg"
}

func msgType(code uint64) string {
	return msgTypesDict[code]
}
