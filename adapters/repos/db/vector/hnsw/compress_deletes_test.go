//                           _       _
// __      _____  __ ___   ___  __ _| |_ ___
// \ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
//  \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
//   \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
//
//  Copyright © 2016 - 2023 Weaviate B.V. All rights reserved.
//
//  CONTACT: hello@weaviate.io
//

//go:build !race

package hnsw_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaviate/weaviate/adapters/repos/db/lsmkv"
	"github.com/weaviate/weaviate/adapters/repos/db/vector/common"
	"github.com/weaviate/weaviate/adapters/repos/db/vector/hnsw"
	"github.com/weaviate/weaviate/adapters/repos/db/vector/hnsw/distancer"
	"github.com/weaviate/weaviate/adapters/repos/db/vector/ssdhelpers"
	"github.com/weaviate/weaviate/adapters/repos/db/vector/testinghelpers"
	"github.com/weaviate/weaviate/entities/cyclemanager"
	ent "github.com/weaviate/weaviate/entities/vectorindex/hnsw"
)

func Test_NoRaceCompressDoesNotCrash(t *testing.T) {
	efConstruction := 64
	ef := 32
	maxNeighbors := 32
	dimensions := 20
	vectors_size := 10000
	queries_size := 100
	k := 100
	delete_indices := make([]uint64, 0, 1000)
	for i := 0; i < 1000; i++ {
		delete_indices = append(delete_indices, uint64(i+10))
	}
	delete_indices = append(delete_indices, uint64(1))

	vectors, queries := testinghelpers.RandomVecs(vectors_size, queries_size, dimensions)
	distancer := distancer.NewL2SquaredProvider()

	uc := ent.UserConfig{}
	uc.MaxConnections = maxNeighbors
	uc.EFConstruction = efConstruction
	uc.EF = ef
	uc.VectorCacheMaxObjects = 10e12
	uc.PQ = ent.PQConfig{Enabled: true, Encoder: ent.PQEncoder{Type: "title", Distribution: "normal"}}

	index, _ := hnsw.New(hnsw.Config{
		RootPath:              t.TempDir(),
		ID:                    "recallbenchmark",
		MakeCommitLoggerThunk: hnsw.MakeNoopCommitLogger,
		DistanceProvider:      distancer,
		VectorForIDThunk: func(ctx context.Context, id uint64) ([]float32, error) {
			return vectors[int(id)], nil
		},
		TempVectorForIDThunk: func(ctx context.Context, id uint64, container *common.VectorSlice) ([]float32, error) {
			copy(container.Slice, vectors[int(id)])
			return container.Slice, nil
		},
	}, uc, cyclemanager.NewCallbackGroupNoop(), cyclemanager.NewCallbackGroupNoop(),
		cyclemanager.NewCallbackGroupNoop(), newDummyStore(t))
	defer index.Shutdown(context.Background())
	ssdhelpers.Concurrently(uint64(len(vectors)), func(id uint64) {
		index.Add(uint64(id), vectors[id])
	})
	index.Delete(delete_indices...)

	cfg := ent.PQConfig{
		Enabled: true,
		Encoder: ent.PQEncoder{
			Type:         ent.PQEncoderTypeKMeans,
			Distribution: ent.PQEncoderDistributionLogNormal,
		},
		Segments:  dimensions,
		Centroids: 256,
	}
	index.Compress(cfg)
	for _, v := range queries {
		_, _, err := index.SearchByVector(v, k, nil)
		assert.Nil(t, err)
	}
}

func TestHnswPqNilVectors(t *testing.T) {
	dimensions := 20
	vectors_size := 10_000
	queries_size := 10

	vectors, _ := testinghelpers.RandomVecs(vectors_size, queries_size, dimensions)

	// set some vectors to nil
	for i := range vectors {
		if i == 500 {
			vectors[i] = nil
		}
	}

	userConfig := ent.UserConfig{
		MaxConnections: 30,
		EFConstruction: 64,
		EF:             32,

		// The actual size does not matter for this test, but if it defaults to
		// zero it will constantly think it's full and needs to be deleted - even
		// after just being deleted, so make sure to use a positive number here.
		VectorCacheMaxObjects: 1000000,
	}

	rootPath := "doesnt-matter-as-committlogger-is-mocked-out"
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			fmt.Println(err)
		}
	}(rootPath)

	index, err := hnsw.New(hnsw.Config{
		RootPath:              rootPath,
		ID:                    "nil-vector-test",
		MakeCommitLoggerThunk: hnsw.MakeNoopCommitLogger,
		DistanceProvider:      distancer.NewCosineDistanceProvider(),
		VectorForIDThunk: func(ctx context.Context, id uint64) ([]float32, error) {
			return vectors[int(id)], nil
		},
		TempVectorForIDThunk: hnsw.TempVectorForIDThunk(vectors),
	}, userConfig, cyclemanager.NewCallbackGroupNoop(), cyclemanager.NewCallbackGroupNoop(), cyclemanager.NewCallbackGroupNoop(), nil)

	require.NoError(t, err)

	ssdhelpers.Concurrently(uint64(len(vectors)/2), func(id uint64) {
		if vectors[id] == nil {
			return
		}

		err := index.Add(uint64(id), vectors[id])
		require.Nil(t, err)
	})

	userConfig.PQ = ent.PQConfig{
		Enabled: true,
		Encoder: ent.PQEncoder{
			Type:         ent.PQEncoderTypeTile,
			Distribution: ent.PQEncoderDistributionLogNormal,
		},
		BitCompression: false,
		Segments:       0,
		Centroids:      256,
	}

	ch := make(chan error)
	err = index.UpdateUserConfig(userConfig, func() {
		close(ch)
	})
	require.NoError(t, err)

	<-ch
	start := uint64(len(vectors) / 2)
	ssdhelpers.Concurrently(uint64(len(vectors)/2), func(id uint64) {
		if vectors[id+start] == nil {
			return
		}

		err = index.Add(uint64(id)+start, vectors[id+start])
		require.Nil(t, err)
	})
}

func newDummyStore(t testing.TB) *lsmkv.Store {
	logger, _ := test.NewNullLogger()
	storeDir := t.TempDir()
	store, err := lsmkv.New(storeDir, storeDir, logger, nil,
		cyclemanager.NewCallbackGroupNoop(), cyclemanager.NewCallbackGroupNoop())
	require.Nil(t, err)
	return store
}
