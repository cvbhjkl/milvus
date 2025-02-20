// Copyright 2023 yah01
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package segments

import (
	"context"
	"sync"

	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/milvus-io/milvus-proto/go-api/v2/msgpb"
	"github.com/milvus-io/milvus/internal/proto/datapb"
	"github.com/milvus-io/milvus/internal/proto/segcorepb"
	storage "github.com/milvus-io/milvus/internal/storage"
	"github.com/milvus-io/milvus/pkg/log"
	"github.com/milvus-io/milvus/pkg/util/merr"
	"github.com/milvus-io/milvus/pkg/util/typeutil"
)

var _ Segment = (*L0Segment)(nil)

type L0Segment struct {
	baseSegment

	dataGuard sync.RWMutex
	pks       []storage.PrimaryKey
	tss       []uint64
}

func NewL0Segment(collection *Collection,
	segmentID int64,
	partitionID int64,
	collectionID int64,
	shard string,
	segmentType SegmentType,
	version int64,
	startPosition *msgpb.MsgPosition,
	deltaPosition *msgpb.MsgPosition,
) (Segment, error) {
	/*
		CSegmentInterface
		NewSegment(CCollection collection, uint64_t segment_id, SegmentType seg_type);
	*/

	log.Info("create L0 segment",
		zap.Int64("collectionID", collectionID),
		zap.Int64("partitionID", partitionID),
		zap.Int64("segmentID", segmentID),
		zap.String("segmentType", segmentType.String()))

	segment := &L0Segment{
		baseSegment: newBaseSegment(segmentID, partitionID, collectionID, shard, segmentType, version, startPosition),
	}

	return segment, nil
}

func (s *L0Segment) RLock() error {
	return nil
}

func (s *L0Segment) RUnlock() {}

func (s *L0Segment) InsertCount() int64 {
	return 0
}

func (s *L0Segment) RowNum() int64 {
	return 0
}

func (s *L0Segment) MemSize() int64 {
	return lo.SumBy(s.pks, func(pk storage.PrimaryKey) int64 {
		return pk.Size() + 8
	})
}

func (s *L0Segment) LastDeltaTimestamp() uint64 {
	s.dataGuard.RLock()
	defer s.dataGuard.RUnlock()

	last, err := lo.Last(s.tss)
	if err != nil {
		return 0
	}
	return last
}

func (s *L0Segment) GetIndex(fieldID int64) *IndexedFieldInfo {
	return nil
}

func (s *L0Segment) ExistIndex(fieldID int64) bool {
	return false
}

func (s *L0Segment) HasRawData(fieldID int64) bool {
	return false
}

func (s *L0Segment) Indexes() []*IndexedFieldInfo {
	return nil
}

func (s *L0Segment) Type() SegmentType {
	return s.typ
}

func (s *L0Segment) Level() datapb.SegmentLevel {
	return datapb.SegmentLevel_L0
}

func (s *L0Segment) Search(ctx context.Context, searchReq *SearchRequest) (*SearchResult, error) {
	return nil, nil
}

func (s *L0Segment) Retrieve(ctx context.Context, plan *RetrievePlan) (*segcorepb.RetrieveResults, error) {
	return nil, nil
}

func (s *L0Segment) Insert(ctx context.Context, rowIDs []int64, timestamps []typeutil.Timestamp, record *segcorepb.InsertRecord) error {
	return merr.WrapErrIoFailedReason("insert not supported for L0 segment")
}

func (s *L0Segment) Delete(ctx context.Context, primaryKeys []storage.PrimaryKey, timestamps []typeutil.Timestamp) error {
	return merr.WrapErrIoFailedReason("delete not supported for L0 segment")
}

func (s *L0Segment) LoadDeltaData(ctx context.Context, deltaData *storage.DeleteData) error {
	s.dataGuard.Lock()
	defer s.dataGuard.Unlock()

	s.pks = append(s.pks, deltaData.Pks...)
	s.tss = append(s.tss, deltaData.Tss...)
	return nil
}

func (s *L0Segment) DeleteRecords() ([]storage.PrimaryKey, []uint64) {
	s.dataGuard.RLock()
	defer s.dataGuard.RUnlock()

	return s.pks, s.tss
}

func (s *L0Segment) Release() {
	s.dataGuard.Lock()
	defer s.dataGuard.Unlock()

	s.pks = nil
	s.tss = nil
}
