// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/productcatalogservice/genproto"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type productCatalog struct {
	pb.UnimplementedProductCatalogServiceServer
	catalog    pb.ListProductsResponse
	snapshot   atomic.Pointer[catalogSnapshot]
	snapshotMu sync.Mutex
}

func (p *productCatalog) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (p *productCatalog) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (p *productCatalog) ListProducts(context.Context, *pb.Empty) (*pb.ListProductsResponse, error) {
	time.Sleep(extraLatency)

	return &pb.ListProductsResponse{Products: p.parseCatalog()}, nil
}

func (p *productCatalog) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	time.Sleep(extraLatency)

	snapshot := p.catalogSnapshot()
	var found *pb.Product
	if snapshot != nil {
		found = snapshot.byID[req.Id]
	}

	if found == nil {
		return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
	}
	return found, nil
}

func (p *productCatalog) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	time.Sleep(extraLatency)

	var ps []*pb.Product
	for _, product := range p.parseCatalog() {
		if strings.Contains(strings.ToLower(product.Name), strings.ToLower(req.Query)) ||
			strings.Contains(strings.ToLower(product.Description), strings.ToLower(req.Query)) {
			ps = append(ps, product)
		}
	}

	return &pb.SearchProductsResponse{Results: ps}, nil
}

// A published snapshot and its products are immutable. Reloads replace the
// complete snapshot so concurrent readers never observe a partially built index.
type catalogSnapshot struct {
	products []*pb.Product
	byID     map[string]*pb.Product
}

func (p *productCatalog) catalogSnapshot() *catalogSnapshot {
	reload := reloadCatalog.Load()
	if snapshot := p.snapshot.Load(); snapshot != nil && !reload {
		return snapshot
	}
	p.snapshotMu.Lock()
	defer p.snapshotMu.Unlock()
	if snapshot := p.snapshot.Load(); snapshot != nil && !reload {
		return snapshot
	}
	products := p.catalog.Products
	if reload || len(products) == 0 {
		var fresh pb.ListProductsResponse
		if err := loadCatalog(&fresh); err != nil {
			return nil
		}
		products = fresh.Products
	}
	snapshot := &catalogSnapshot{products: products, byID: make(map[string]*pb.Product, len(products))}
	for _, product := range products {
		// Preserve the original scan's last-match behavior for duplicate IDs.
		snapshot.byID[product.Id] = product
	}
	p.snapshot.Store(snapshot)
	return snapshot
}

func (p *productCatalog) parseCatalog() []*pb.Product {
	if snapshot := p.catalogSnapshot(); snapshot != nil {
		return snapshot.products
	}
	return []*pb.Product{}
}
