// The sl module has been writen as a lightweight replacement for the C libslink library.
// It is aimed at clients that need to connect and decode data from a seedlink server.
//
// The seedlink code is not a direct replacement for libslink. It can run in two modes, either as a
// raw connection to the client connection (Conn) which allows mechanisms to monitor or have a finer
// control of the SeedLink connection, or in the collection mode (SLink) where a connection is established
// and received miniseed blocks can be processed with a call back function. A context can be passed into
// the collection loop to allow interuption or as a shutdown mechanism. It is not passed to the underlying
// seedlink connection messaging which is managed via a deadline mechanism, e.g. the `SetTimeout` option.
//
// An example Seedlink application can be as simple as:
//
//  if err := sl.NewSLink().Collect(func(seq string, data []byte) (bool, error) {
//	   //... process miniseed data
//
//         return false, nil
//  }); err != nil {
//          log.Fatal(err)
//  }
//
// A state mechanism is available for the initial connection, although it is the clients responsibility to
// periodically maintain its content.
//
package sl
