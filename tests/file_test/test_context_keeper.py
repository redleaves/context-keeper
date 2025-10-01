#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Context-Keeper 代码上下文管理功能测试脚本
"""

import os
import subprocess
import json
import time
import sys
import random
import string

# 测试会话ID (使用实际会话ID或者设置为None自动创建)
SESSION_ID = "session-20250401-160421"

# 测试文件路径
TEST_FILE_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), "test_file.py")

def run_cmd(cmd):
    """执行命令并返回结果"""
    process = subprocess.Popen(
        cmd, 
        shell=True, 
        stdout=subprocess.PIPE, 
        stderr=subprocess.PIPE
    )
    stdout, stderr = process.communicate()
    return {
        "stdout": stdout.decode("utf-8"),
        "stderr": stderr.decode("utf-8"),
        "returncode": process.returncode
    }

def generate_random_content():
    """生成随机代码内容"""
    functions = []
    for i in range(3):
        func_name = f"func_{i}_{''.join(random.choices(string.ascii_lowercase, k=5))}"
        func_content = f"""
def {func_name}(param1, param2):
    \"\"\"测试函数 {i}\"\"\"
    result = param1 + param2
    print(f"计算结果: {result}")
    return result
"""
        functions.append(func_content)
    
    return f"""#!/usr/bin/env python3
# -*- coding: utf-8 -*-

\"\"\"
自动生成的测试文件
\"\"\"

{' '.join(functions)}

def main():
    print("测试程序启动")
    
if __name__ == "__main__":
    main()
"""

def create_test_file():
    """创建测试文件"""
    content = generate_random_content()
    with open(TEST_FILE_PATH, "w") as f:
        f.write(content)
    print(f"[+] 创建测试文件: {TEST_FILE_PATH}")
    return content

def modify_test_file():
    """修改测试文件"""
    # 读取现有内容
    with open(TEST_FILE_PATH, "r") as f:
        content = f.read()
    
    # 添加新函数
    new_func = f"""
def added_function_{int(time.time())}():
    \"\"\"新添加的函数\"\"\"
    print("这是一个新添加的函数")
    return "新函数"
"""
    
    # 在main函数前插入新函数
    if "def main():" in content:
        content = content.replace("def main():", new_func + "\ndef main():")
    else:
        content += new_func
    
    # 写回文件
    with open(TEST_FILE_PATH, "w") as f:
        f.write(content)
    
    print(f"[+] 修改测试文件: {TEST_FILE_PATH}")
    return content

def test_file_association():
    """测试文件关联功能"""
    print("\n===== 测试文件关联功能 =====")
    
    # 获取会话ID
    session_id = SESSION_ID
    if not session_id:
        print("[*] 创建新会话...")
        cmd = "curl -s -X POST 'http://localhost:8080/api/sessions' -H 'Content-Type: application/json' -d '{\"action\":\"create\"}'"
        result = run_cmd(cmd)
        try:
            session_id = json.loads(result["stdout"])["sessionId"]
            print(f"[+] 新会话创建成功: {session_id}")
        except:
            print(f"[-] 创建会话失败: {result}")
            return False
    
    # 关联文件
    print(f"[*] 关联文件: {TEST_FILE_PATH} 到会话: {session_id}")
    cmd = f"curl -s -X POST 'http://localhost:8080/api/files/associate' -H 'Content-Type: application/json' -d '{{\"sessionId\":\"{session_id}\",\"filePath\":\"{TEST_FILE_PATH.replace('\\', '\\\\')}\"}}'"
    result = run_cmd(cmd)
    
    if "成功关联文件" in result["stdout"] or "success" in result["stdout"].lower():
        print(f"[+] 文件关联成功: {result['stdout']}")
        return True
    else:
        print(f"[-] 文件关联失败: {result}")
        return False

def test_record_edit():
    """测试编辑记录功能"""
    print("\n===== 测试编辑记录功能 =====")
    
    # 获取会话ID
    session_id = SESSION_ID
    if not session_id:
        print("[-] 没有有效会话ID，无法测试编辑记录")
        return False
    
    # 修改文件
    original_content = None
    with open(TEST_FILE_PATH, "r") as f:
        original_content = f.read()
    
    new_content = modify_test_file()
    
    # 计算diff
    import difflib
    diff = "\n".join(difflib.unified_diff(
        original_content.splitlines(),
        new_content.splitlines(),
        fromfile='a/' + os.path.basename(TEST_FILE_PATH),
        tofile='b/' + os.path.basename(TEST_FILE_PATH),
        lineterm=''
    ))
    
    # 记录编辑
    print(f"[*] 记录编辑操作 到会话: {session_id}")
    cmd = f"curl -s -X POST 'http://localhost:8080/api/edits/record' -H 'Content-Type: application/json' -d '{{\"sessionId\":\"{session_id}\",\"filePath\":\"{TEST_FILE_PATH.replace('\\', '\\\\')}\",\"diff\":{json.dumps(diff)}}}'"
    result = run_cmd(cmd)
    
    if "成功记录编辑操作" in result["stdout"] or "success" in result["stdout"].lower():
        print(f"[+] 编辑记录成功: {result['stdout']}")
        return True
    else:
        print(f"[-] 编辑记录失败: {result}")
        return False

def test_programming_context():
    """测试编程上下文获取功能"""
    print("\n===== 测试编程上下文获取功能 =====")
    
    # 获取会话ID
    session_id = SESSION_ID
    if not session_id:
        print("[-] 没有有效会话ID，无法测试编程上下文获取")
        return False
    
    # 获取编程上下文
    print(f"[*] 获取编程上下文 从会话: {session_id}")
    cmd = f"curl -s -X POST 'http://localhost:8080/api/context/programming' -H 'Content-Type: application/json' -d '{{\"sessionId\":\"{session_id}\",\"query\":\"测试文件\"}}'"
    result = run_cmd(cmd)
    
    try:
        # 尝试解析JSON
        response = json.loads(result["stdout"])
        
        # 验证返回的是ProgrammingContext格式
        if "sessionId" in response:
            print(f"[+] 编程上下文获取成功")
            
            # 打印关联文件数量
            if "associatedFiles" in response and len(response["associatedFiles"]) > 0:
                print(f"[+] 关联文件数量: {len(response['associatedFiles'])}")
                # 随机打印一个文件的详细信息
                sample_file = response["associatedFiles"][0]
                print(f"[+] 示例文件: {sample_file['path']}")
                if "relatedDiscussions" in sample_file and len(sample_file["relatedDiscussions"]) > 0:
                    print(f"[+] 相关讨论数量: {len(sample_file['relatedDiscussions'])}")
            else:
                print(f"[*] 没有关联文件")
                
            # 打印编辑历史数量
            if "recentEdits" in response and len(response["recentEdits"]) > 0:
                print(f"[+] 编辑历史数量: {len(response['recentEdits'])}")
            
            # 打印代码片段数量
            if "relevantSnippets" in response and len(response["relevantSnippets"]) > 0:
                print(f"[+] 相关代码片段数量: {len(response['relevantSnippets'])}")
            
            return True
        else:
            print(f"[-] 编程上下文格式异常: {response}")
            return False
    except Exception as e:
        print(f"[-] 编程上下文获取失败: {result}")
        print(f"[-] 异常: {e}")
        return False

def main():
    """主测试函数"""
    print("==================================")
    print("Context-Keeper 代码上下文管理测试脚本")
    print("==================================\n")
    
    # 创建测试文件
    create_test_file()
    
    # 测试文件关联
    association_result = test_file_association()
    
    # 测试编辑记录
    if association_result:
        edit_result = test_record_edit()
    else:
        print("[-] 由于文件关联失败，跳过编辑记录测试")
        edit_result = False
    
    # 测试编程上下文获取
    if association_result:
        context_result = test_programming_context()
    else:
        print("[-] 由于文件关联失败，跳过编程上下文测试")
        context_result = False
    
    # 总结测试结果
    print("\n==================================")
    print("测试结果总结:")
    print(f"文件关联测试: {'✅ 通过' if association_result else '❌ 失败'}")
    print(f"编辑记录测试: {'✅ 通过' if edit_result else '❌ 失败'}")
    print(f"上下文获取测试: {'✅ 通过' if context_result else '❌ 失败'}")
    print("==================================")

if __name__ == "__main__":
    main() 