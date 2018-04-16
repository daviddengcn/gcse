package main

import (
	"context"
	"flag"
	"net"

	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/gcse/store"
	"github.com/daviddengcn/gcse/utils"
	"github.com/golang/glog"
	"google.golang.org/grpc"

	gpb "github.com/daviddengcn/gcse/shared/proto"
)

type server struct {
}

var _ gpb.StoreServiceServer = (*server)(nil)

func (s *server) PackageCrawlHistory(_ context.Context, req *gpb.PackageCrawlHistoryReq) (*gpb.PackageCrawlHistoryResp, error) {
	site, path := utils.SplitPackage(req.Package)
	info, err := store.ReadPackageHistory(site, path)
	if err != nil {
		glog.Errorf("ReadPackageHistoryOf %q %q failed: %v", site, path, err)
		return nil, err
	}
	return &gpb.PackageCrawlHistoryResp{Info: info}, nil
}

func main() {
	addr := flag.String("addr", configs.StoreDAddr, "addr to listen")

	flag.Parse()

	glog.Infof("Listening to %s", *addr)
	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	gpb.RegisterStoreServiceServer(grpcServer, &server{})
	grpcServer.Serve(lis)
}
