#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
测试文件 - 用于验证Context-Keeper代码上下文管理功能
"""

import json
import os
import sys
import time
import subprocess

# 运行命令的通用函数
def run_cmd(cmd):
    print(f"执行命令: {cmd}")
    try:
        process = subprocess.Popen(
            cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
        )
        stdout, stderr = process.communicate()
        return {
            "code": process.returncode,
            "stdout": stdout.decode("utf-8", errors="ignore"),
            "stderr": stderr.decode("utf-8", errors="ignore"),
        }
    except Exception as e:
        return {"code": -1, "stdout": "", "stderr": str(e)}

def hello_world():
    """简单的问候函数"""
    print("Hello, Context-Keeper!")
    return "Hello, World!"

def calculate_sum(a, b):
    """计算两个数的和"""
    return a + b

def multiply(a, b):
    """计算两个数的乘积"""
    return a * b

class TestClass:
    """测试类"""
    
    def __init__(self, name):
        self.name = name
        
    def greet(self):
        """打招呼方法"""
        return f"Hello, {self.name}!"
    
    def goodbye(self):
        """告别方法"""
        return f"Goodbye, {self.name}!"

def main():
    # 测试基本函数
    result = hello_world()
    print(f"函数返回值: {result}")
    
    # 测试计算函数
    sum_result = calculate_sum(10, 20)
    product = multiply(5, 7)
    print(f"求和结果: {sum_result}")
    print(f"乘积结果: {product}")
    
    # 测试类
    test_obj = TestClass("Context-Keeper")
    greeting = test_obj.greet()
    farewell = test_obj.goodbye()
    print(f"类方法返回值: {greeting}")
    print(f"告别消息: {farewell}")

# 测试编程上下文获取功能
def test_programming_context_api():
    """测试编程上下文API调用"""
    print("\n===== 测试编程上下文API调用 =====")
    
    # 先创建一个会话
    print("[*] 创建新会话")
    cmd = "curl -s -X POST 'http://localhost:8080/api/context/session/create' -H 'Content-Type: application/json' -d '{}'"
    result = run_cmd(cmd)
    
    if result["code"] != 0:
        print(f"[-] 创建会话失败: {result['stderr']}")
        return False
    
    try:
        session_response = json.loads(result["stdout"])
        session_id = session_response.get("sessionId", "")
        if not session_id:
            print("[-] 无法获取会话ID")
            return False
        
        print(f"[+] 创建会话成功: {session_id}")
        
        # 关联一个代码文件
        test_file = "tests/file_test/test_code.py"
        print(f"[*] 关联代码文件: {test_file}")
        
        cmd = f"curl -s -X POST 'http://localhost:8080/api/context/associate' -H 'Content-Type: application/json' -d '{{\"sessionId\":\"{session_id}\",\"filePath\":\"{test_file}\"}}'"
        result = run_cmd(cmd)
        
        if result["code"] != 0 or "error" in result["stdout"].lower():
            print(f"[-] 关联文件失败: {result['stdout']}")
            return False
        
        print(f"[+] 关联文件成功")
        
        # 获取编程上下文
        print(f"[*] 获取编程上下文")
        cmd = f"curl -s -X POST 'http://localhost:8080/api/context/programming' -H 'Content-Type: application/json' -d '{{\"sessionId\":\"{session_id}\",\"query\":\"test_code\"}}'"
        result = run_cmd(cmd)
        
        if result["code"] != 0:
            print(f"[-] 获取编程上下文失败: {result['stderr']}")
            return False
        
        context_response = json.loads(result["stdout"])
        print(f"[+] 获取编程上下文成功:")
        print(json.dumps(context_response, indent=2, ensure_ascii=False)[:300] + "...")
        
        # 验证上下文内容
        if "associatedFiles" in context_response and len(context_response["associatedFiles"]) > 0:
            print(f"[+] 关联的文件数量: {len(context_response['associatedFiles'])}")
        else:
            print("[-] 未找到关联文件信息")
        
        return True
        
    except Exception as e:
        print(f"[-] 测试出错: {str(e)}")
        return False

if __name__ == "__main__":
    # 运行测试
    main()
    test_programming_context_api() 