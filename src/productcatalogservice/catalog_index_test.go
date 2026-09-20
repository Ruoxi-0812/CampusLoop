package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/productcatalogservice/genproto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIndexDuplicateAndMissingIDs(t *testing.T) {
	svc := &productCatalog{catalog: pb.ListProductsResponse{Products: []*pb.Product{
		{Id: "duplicate", Name: "first"}, {Id: "duplicate", Name: "last"},
	}}}
	got, err := svc.GetProduct(context.Background(), &pb.GetProductRequest{Id: "duplicate"})
	if err != nil || got.GetName() != "last" {
		t.Fatalf("last match changed: %v, %v", got, err)
	}
	_, err = svc.GetProduct(context.Background(), &pb.GetProductRequest{Id: "absent"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("missing ID: %v", err)
	}
}

func TestIndexConcurrentInitialization(t *testing.T) {
	svc := &productCatalog{catalog: pb.ListProductsResponse{Products: []*pb.Product{{Id: "one", Name: "One"}}}}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				got, err := svc.GetProduct(context.Background(), &pb.GetProductRequest{Id: "one"})
				if err != nil || got.GetName() != "One" {
					t.Errorf("lookup: %v %v", got, err)
					return
				}
				listed, err := svc.ListProducts(context.Background(), &pb.Empty{})
				if err != nil || len(listed.GetProducts()) != 1 {
					t.Errorf("list: %v %v", listed, err)
					return
				}
				searched, err := svc.SearchProducts(context.Background(), &pb.SearchProductsRequest{Query: "one"})
				if err != nil || len(searched.GetResults()) != 1 {
					t.Errorf("search: %v %v", searched, err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestIndexReloadAndRecovery(t *testing.T) {
	t.Setenv("ALLOYDB_CLUSTER_NAME", "")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		reloadCatalog.Store(false)
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	})
	write := func(data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "products.json"), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"products":[{"id":"old","name":"Old"}]}`)
	svc := &productCatalog{}
	old := svc.parseCatalog()
	if len(old) != 1 || old[0].Id != "old" {
		t.Fatalf("initial load: %v", old)
	}
	write(`{"products":[{"id":"new","name":"New"}]}`)
	reloadCatalog.Store(true)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := svc.GetProduct(context.Background(), &pb.GetProductRequest{Id: "new"})
			if err != nil || got.GetName() != "New" {
				t.Errorf("reload: %v %v", got, err)
			}
		}()
	}
	wg.Wait()
	reloadCatalog.Store(false)
	if old[0].Id != "old" {
		t.Fatal("reload mutated an earlier snapshot")
	}
	_, err = svc.GetProduct(context.Background(), &pb.GetProductRequest{Id: "old"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("stale index entry: %v", err)
	}
	if got := svc.parseCatalog(); len(got) != 1 || got[0].Id != "new" {
		t.Fatalf("stale list: %v", got)
	}
	write(`invalid json`)
	reloadCatalog.Store(true)
	if got := svc.parseCatalog(); len(got) != 0 {
		t.Fatalf("failed reload should return empty: %v", got)
	}
	write(`{"products":[{"id":"recovered","name":"Recovered"}]}`)
	got, err := svc.GetProduct(context.Background(), &pb.GetProductRequest{Id: "recovered"})
	if err != nil || got.GetName() != "Recovered" {
		t.Fatalf("recovery: %v %v", got, err)
	}
}
