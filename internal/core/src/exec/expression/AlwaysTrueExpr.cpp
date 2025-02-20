// Licensed to the LF AI & Data foundation under one
// or more contributor license agreements. See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership. The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include "AlwaysTrueExpr.h"

namespace milvus {
namespace exec {

void
PhyAlwaysTrueExpr::Eval(EvalCtx& context, VectorPtr& result) {
    int64_t real_batch_size = current_pos_ + batch_size_ >= num_rows_
                                  ? num_rows_ - current_pos_
                                  : batch_size_;

    if (real_batch_size == 0) {
        result = nullptr;
        return;
    }

    auto res_vec =
        std::make_shared<ColumnVector>(DataType::BOOL, real_batch_size);
    bool* res_bool = (bool*)res_vec->GetRawData();
    for (size_t i = 0; i < real_batch_size; ++i) {
        res_bool[i] = true;
    }

    result = res_vec;
    current_pos_ += real_batch_size;
}

}  //namespace exec
}  // namespace milvus
