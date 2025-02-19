// Copyright (C) 2019-2020 Zilliz. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License
// is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied. See the License for the specific language governing permissions and limitations under the License

#include <gtest/gtest.h>
#include <random>
#include "segcore/InsertRecord.h"

using namespace milvus;
using namespace milvus::segcore;

template <typename T>
class TypedOffsetOrderedArrayTest : public testing::Test {
 public:
    void
    SetUp() override {
        er = std::default_random_engine(42);
    }

    void
    TearDown() override {
    }

 protected:
    void
    insert(T pk) {
        map_.insert(pk, offset_++);
        data_.push_back(pk);
        std::sort(data_.begin(), data_.end());
    }

    void
    seal() {
        map_.seal();
    }

    std::vector<T>
    random_generate(int num) {
        std::vector<T> res;
        for (int i = 0; i < num; i++) {
            if constexpr (std::is_same_v<std::string, T>) {
                res.push_back(std::to_string(er()));
            } else {
                res.push_back(static_cast<T>(er()));
            }
        }
        return res;
    }

 protected:
    int64_t offset_ = 0;
    std::vector<T> data_;
    milvus::segcore::OffsetOrderedArray<T> map_;
    std::default_random_engine er;
};

using TypeOfPks = testing::Types<int64_t, std::string>;
TYPED_TEST_CASE_P(TypedOffsetOrderedArrayTest);

TYPED_TEST_P(TypedOffsetOrderedArrayTest, find_first) {
    std::vector<int64_t> offsets;

    // not sealed.
    ASSERT_ANY_THROW(this->map_.find_first(Unlimited, {}, true));

    // insert 10 entities.
    int num = 10;
    auto data = this->random_generate(num);
    for (const auto& x : data) {
        this->insert(x);
    }

    // seal.
    this->seal();

    // all is satisfied.
    BitsetType all(num);
    all.set();
    offsets = this->map_.find_first(num / 2, all, true);
    ASSERT_EQ(num / 2, offsets.size());
    for (int i = 1; i < offsets.size(); i++) {
        ASSERT_TRUE(data[offsets[i - 1]] <= data[offsets[i]]);
    }
    offsets = this->map_.find_first(Unlimited, all, true);
    ASSERT_EQ(num, offsets.size());
    for (int i = 1; i < offsets.size(); i++) {
        ASSERT_TRUE(data[offsets[i - 1]] <= data[offsets[i]]);
    }

    // corner case, segment offset exceeds the size of bitset.
    BitsetType all_minus_1(num - 1);
    all_minus_1.set();
    offsets = this->map_.find_first(num / 2, all_minus_1, true);
    ASSERT_EQ(num / 2, offsets.size());
    for (int i = 1; i < offsets.size(); i++) {
        ASSERT_TRUE(data[offsets[i - 1]] <= data[offsets[i]]);
    }
    offsets = this->map_.find_first(Unlimited, all_minus_1, true);
    ASSERT_EQ(all_minus_1.size(), offsets.size());
    for (int i = 1; i < offsets.size(); i++) {
        ASSERT_TRUE(data[offsets[i - 1]] <= data[offsets[i]]);
    }

    // none is satisfied.
    BitsetType none(num);
    none.reset();
    offsets = this->map_.find_first(num / 2, none, true);
    ASSERT_EQ(0, offsets.size());
    offsets = this->map_.find_first(NoLimit, none, true);
    ASSERT_EQ(0, offsets.size());
}

REGISTER_TYPED_TEST_CASE_P(TypedOffsetOrderedArrayTest, find_first);
INSTANTIATE_TYPED_TEST_CASE_P(Prefix, TypedOffsetOrderedArrayTest, TypeOfPks);
