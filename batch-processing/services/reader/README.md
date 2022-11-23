The reader service is a gRPC implementation to handle the following: 

* Client that can be invoked via cmd that will open a file and stream it to the server via gRPC.
* Server will receive the stream and publish rows to kafka
* Server will be always on, client is invokable.