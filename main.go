package main

import (
	"io"
	"time"
	"sync"
	"context"
	"fmt"
	"net"
	"go-grpc/logger"

	"google.golang.org/grpc"
)

type customLogger struct {
	logger.UnimplementedLoggerServer
	Store concurrentStorage
}

type concurrentStorage struct {
	sync.Mutex
	Data []*logger.LogEntry
}

func (c *customLogger) SingleLogEntry(ctx context.Context, req *logger.LogEntry) (*logger.SingleLogResponse, error) {
	c.Store.Lock()
	c.Store.Data = append(c.Store.Data, req)
	c.Store.Unlock()
	time.Sleep(2 * time.Second)
	fmt.Println("SingleLogEntry: finished logging...")
	return &logger.SingleLogResponse {
		NumberOfWrittenChars: int32(len(req.GetMessage())),
	}, nil
}

func (c *customLogger) BatchLogEntry(stream logger.Logger_BatchLogEntryServer) error {
	res := &logger.BatchLogResponse{}
	for {
		logEntry, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("BatchLogEntry: finished logging...")
			stream.SendAndClose(res)
		}
		if err != nil {
			return err
		}
		res.NumberOfWrittenMessages++
		res.NumberOfWrittenChars += int32(len(logEntry.GetMessage()))
		c.Store.Lock()
		c.Store.Data = append(c.Store.Data, logEntry)
		c.Store.Unlock()
		time.Sleep(time.Second * 2)
	}
}

func (c *customLogger) GetLogsByApp(req *logger.App, stream logger.Logger_GetLogsByAppServer) error {
	for _, v := range c.Store.Data {
		if v.GetAppName() == req.GetName() {
			if err := stream.Send(v); err != nil {
				return err
			}
			time.Sleep(time.Second * 2)
		}
	}
	fmt.Println("GetLogsByApp: finished sending messages...")
	return nil
}

func main() {
	grpcServer := grpc.NewServer()
	myLogger := &customLogger{
		Store: concurrentStorage{
			Data: make([]*logger.LogEntry, 0),
		},
	}
	logger.RegisterLoggerServer(grpcServer, myLogger)
	lis, err := net.Listen("tcp", ":8089")
	if err != nil {
		panic(err)
	}
	if err := grpcServer.Serve(lis); err != nil {
		panic(err)
	}
}
