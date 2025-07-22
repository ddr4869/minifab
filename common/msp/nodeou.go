package msp

type FabricOUIdentifier struct {
	Certificate                  []byte
	OrganizationalUnitIdentifier string
}

type FabricNodeOUs struct {
	Enable              bool
	ClientOUIdentifier  *FabricOUIdentifier
	PeerOUIdentifier    *FabricOUIdentifier
	AdminOUIdentifier   *FabricOUIdentifier
	OrdererOUIdentifier *FabricOUIdentifier
}

type OUIdentifier struct {
	CertifiersIdentifier         []byte
	OrganizationalUnitIdentifier string
}
