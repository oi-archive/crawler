from __future__ import print_function

import typing

import grpc

import api_pb2
import api_pb2_grpc

ID = "example" # TODO: 题库代号
NAME = "Example OJ" # TODO: 题库全称 

info=api_pb2.Info(id=ID,name=NAME)
debug_mode = False

stub=""

# 该组件启动时被调用一次
# TODO: 在此函数中编写初始化代码
def start():
    pass


# 每次更新时被调用
# 返回值：此次要提交更新的文件列表，key表示文件完整路径名，value表示文件内容
# TODO: 在此方法中编写爬虫程序
def update() -> typing.Dict[str,str]:
    pass

# 组件结束运行时被调用
# TODO: 在此方法中执行释放资源等任务
def stop():
    pass


def runUpdate():
    global stub
    stub.Update(api_pb2.UpdateRequest(Info=info,file=update()))


def run():
    global stub
    channel = grpc.insecure_channel('127.0.0.1:27381')
    stub = api_pb2_grpc.APIStub(channel)
    start()
    response = stub.Register(api_pb2.RegisterRequest(Info=info))
    debug_mode=response.debug_mode
    runUpdate()
    


if __name__ == '__main__':
    run()
