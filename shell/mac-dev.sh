#!/bin/bash

# 获取当前脚本的绝对路径
script_path=$(readlink -f "$0")
# 获取脚本所在目录的上一级目录（即项目根目录）
project_root=$(dirname $(dirname "$script_path"))
# 切换到项目根目录
cd "$project_root"

# 创建bin目录（如果不存在）用于存放可执行文件和配置等内容
mkdir -p bin
if [ $? -ne 0 ]; then
    echo "Failed to create bin directory."
    exit 1
fi

# 进入项目根目录下的main目录（假设Go代码的main函数入口文件都在这里）
cd "$project_root/main"

# 拉取依赖包，确保依赖是最新的
go mod tidy

# 编译各个服务器的可执行文件到项目根目录下的bin目录
chatserver_output="../bin/chatserver"
chatserver_package="../main/chatserver.go"
go build -o "$chatserver_output" "$chatserver_package"

gateserver_output="../bin/gateserver"
gateserver_package="../main/gateserver.go"
go build -o "$gateserver_output" "$gateserver_package"

httpserver_output="../bin/httpserver"
httpserver_package="../main/httpserver.go"
go build -o "$httpserver_output" "$httpserver_package"

loginserver_output="../bin/loginserver"
loginserver_package="../main/loginserver.go"
go build -o "$loginserver_output" "$loginserver_package"

slgserver_output="../bin/slgserver"
slgserver_package="../main/slgserver.go"
go build -o "$slgserver_output" "$slgserver_package"

# 回到项目根目录
cd "$project_root"

# 复制data目录（假设包含配置等文件）到bin目录，方便各个服务器读取配置
cp -r data bin/

# 启动各个服务器（在后台运行，你可以根据实际情况调整启动顺序等）
bin/chatserver &
bin/gateserver &
bin/httpserver &
bin/loginserver &
bin/slgserver &

# 输出提示信息告知用户服务器正在后台启动
echo "Servers are starting in the background. You can check the logs in the bin directory if needed."