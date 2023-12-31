package biface

type IConnManager interface {
	Add(IConnection)                                                        // Add connection
	Remove(IConnection)                                                     // Remove connection
	Get(uint64) (IConnection, error)                                        // Get a connection by ConnID
	Get2(string) (IConnection, error)                                       // Get a connection by string ConnID
	Len() int                                                               // Get current number of connections
	ClearAll()                                                              // Remove and stop all connections
	GetAllConnID() []uint64                                                 // Get all connection IDs
	GetAllConnIdStr() []string                                              // Get all string connection IDs
	Range(func(uint64, IConnection, interface{}) error, interface{}) error  // Traverse all connections
	Range2(func(string, IConnection, interface{}) error, interface{}) error // Traverse all connections 2
}
