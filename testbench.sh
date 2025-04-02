#!/bin/bash

# 参数检查
if [ $# -ne 1 ]; then
    echo "Usage: $0 <output_file>"
    exit 1
fi
dd if=/dev/urandom of=data.txt bs=1 count=1000 status=none
OUTPUT_FILE="$1"
TEST_DATA="data.txt"
COUNT=10000


# 记录开始时间
start_time=$(date +%s.%N)

# 核心优化：只打开一次文件描述符
exec 3<>"$OUTPUT_FILE" || { echo "无法打开文件"; exit 1; }

# 循环写入1000次
    # 使用dd写入到已打开的文件描述符3
    dd if="$TEST_DATA" of=/dev/fd/3 bs=1 count=1000 status=none 2>>error.log

# 关闭文件描述符
exec 3>&-

# 计算耗时
end_time=$(date +%s.%N)
elapsed=$(awk "BEGIN {printf \"%.3f\", $end_time - $start_time}")
throughput=$(awk "BEGIN {printf \"%.2f\", ($COUNT * $(stat -c%s "$TEST_DATA")) / ($elapsed * 1024 * 1024)}")

# 结果输出
echo "=============================="
echo "写入测试完成"
echo "文件:      $OUTPUT_FILE"
echo "写入次数:  $COUNT"
echo "总耗时:    ${elapsed}s"
echo "吞吐量:    ${throughput} MB/s"
echo "=============================="

# 数据验证（可选）
echo -n "数据校验: "
# sleep 5
cmp "$TEST_DATA" "$OUTPUT_FILE" && echo "通过" || echo "失败"