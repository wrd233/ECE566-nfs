#!/bin/bash

MOUNT_POINT="/tmp/nfs-mount"
TEST_DIR="$MOUNT_POINT/perf-test"
LOG_FILE="nfs-performance.log"

# 创建测试目录
mkdir -p "$TEST_DIR"

echo "--- NFS Performance Test ---" | tee "$LOG_FILE"
date | tee -a "$LOG_FILE"

# 测试1: 大文件写入
echo "Test 1: Large file write" | tee -a "$LOG_FILE"
time dd if=/dev/zero of="$TEST_DIR/large_file" bs=1M count=100 2>&1 | tee -a "$LOG_FILE"

# 测试2: 大文件读取
echo "Test 2: Large file read" | tee -a "$LOG_FILE"
time dd if="$TEST_DIR/large_file" of=/dev/null bs=1M 2>&1 | tee -a "$LOG_FILE"

# 测试3: 小文件批量创建
echo "Test 3: Small files creation" | tee -a "$LOG_FILE"
time for i in {1..100}; do
    echo "test content" > "$TEST_DIR/small_file_$i"
done 2>&1 | tee -a "$LOG_FILE"

# 测试4: 小文件批量读取
echo "Test 4: Small files read" | tee -a "$LOG_FILE"
time for i in {1..100}; do
    cat "$TEST_DIR/small_file_$i" > /dev/null
done 2>&1 | tee -a "$LOG_FILE"

# 测试5: 目录操作
echo "Test 5: Directory operations" | tee -a "$LOG_FILE"
time for i in {1..20}; do
    mkdir -p "$TEST_DIR/dir_$i"
    for j in {1..5}; do
        touch "$TEST_DIR/dir_$i/file_$j"
    done
    ls -la "$TEST_DIR/dir_$i" > /dev/null
done 2>&1 | tee -a "$LOG_FILE"

# 清理
echo "Cleaning up..." | tee -a "$LOG_FILE"
rm -rf "$TEST_DIR"

echo "Test completed. Results in $LOG_FILE" | tee -a "$LOG_FILE"