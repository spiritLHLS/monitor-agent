import time
import os
import requests
import json
import datetime
from typing import Tuple, List, Dict, Optional
from DrissionPage import ChromiumPage, ChromiumOptions
from cf5s import detect_cf5s, pass_cf5s

API_BASE_URL = "http://ip:port"
GET_TASK_ENDPOINT = "/spiders/getonetask"
HANDLE_TASK_ENDPOINT = "/spiders/handletask"
TOKEN = ""
MAX_TASK_RUNTIME = 600 
MAX_ATTEMPTS = 3 

def init_browser():
    options = ChromiumOptions()
    arguments = [
        "--accept-lang=en-US",
        "--no-first-run",
        "--force-color-profile=srgb",
        "--metrics-recording-only",
        "--password-store=basic",
        "--use-mock-keychain",
        "--export-tagged-pdf",
        "--no-default-browser-check",
        "--enable-features=NetworkService,NetworkServiceInProcess,LoadCryptoTokenExtension,PermuteTLSExtensions",
        "--disable-gpu",
        "--disable-infobars",
        "--disable-extensions",
        "--disable-popup-blocking",
        "--disable-background-mode",
        "--disable-features=FlashDeprecationWarning,EnablePasswordsAccountStorage,PrivacySandboxSettings4",
        "--deny-permission-prompts",
        "--disable-suggestions-ui",
        "--hide-crash-restore-bubble",
        "--window-size=1920,1080",
    ]
    for argument in arguments:
        options.set_argument(argument)
    page = ChromiumPage(addr_or_opts=options)
    return page

def get_task() -> Optional[Dict]:
    try:
        url = f"{API_BASE_URL}{GET_TASK_ENDPOINT}?flag=cf5s"
        payload = {"token": TOKEN}
        headers = {"Content-Type": "application/json"}
        response = requests.post(url, json=payload, headers=headers)
        response_data = response.json()
        if response.status_code == 200 and response_data.get("success"):
            return response_data.get("data")
        else:
            print(f"获取任务失败: {response_data.get('msg')}")
            return None
    except Exception as e:
        print(f"获取任务时出错: {str(e)}")
        return None

def handle_website_crawling(task: Dict) -> Dict:
    result = {
        "token": task["token"],
        "tag": task["tag"],
        "url": task["url"],
        "billing_type": task["billing_type"],
        "crawl_num": task["crawl_num"],
        "req_method": task["req_method"],
        "start_time": datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
        "success": False,
        "runtime": 0,
        "webdata": "",
        "error_msg": ""
    }
    page = None
    for attempt in range(MAX_ATTEMPTS):
        start_time = time.time()
        try:
            if page is None:
                page = init_browser()
            # extra_headers = {}
            # if task.get("extra_header"):
            #     try:
            #         extra_headers = json.loads(task["extra_header"])
            #     except:
            #         print("无法解析extra_header为JSON格式")
            # if extra_headers:
            #     for key, value in extra_headers.items():
            #         page.set_header(key, value)
            print(f"尝试加载（{attempt + 1}/{MAX_ATTEMPTS}）: {task['url']}")
            task_start_time = time.time()
            page.get(task["url"], timeout=MAX_TASK_RUNTIME)
            time.sleep(5)
            if detect_cf5s(page):
                print("检测到cf5s保护，尝试绕过...")
                if not pass_cf5s(page):
                    print("处理cf5s失败")
                    raise Exception("处理cf5s失败")
                else:
                    print("处理cf5s成功")
            if time.time() - task_start_time > MAX_TASK_RUNTIME:
                raise Exception("任务执行超时")
            page_source = page.html
            result["success"] = True
            result["webdata"] = page_source
            break
        except Exception as e:
            error_msg = f"爬取过程中出错（尝试 {attempt + 1}/{MAX_ATTEMPTS}）: {str(e)}"
            print(error_msg)
            result["error_msg"] = error_msg
            if page:
                try:
                    page.quit()
                except:
                    pass
                page = None
            if attempt == MAX_ATTEMPTS - 1:
                result["success"] = False
                result["error_msg"] = f"任务尝试{MAX_ATTEMPTS}次后失败: {str(e)}"
        finally:
            end_time = time.time()
            result["runtime"] = int(end_time - start_time)
    return result, page

def submit_result(result: Dict) -> bool:
    try:
        url = f"{API_BASE_URL}{HANDLE_TASK_ENDPOINT}"
        headers = {"Content-Type": "application/json"}
        response = requests.post(url, json=result, headers=headers)
        response_data = response.json()
        if response.status_code == 200 and response_data.get("success"):
            print(f"结果提交成功: {response_data.get('msg')}")
            return True
        else:
            print(f"结果提交失败: {response_data.get('msg')}")
            return False
    except Exception as e:
        print(f"提交结果时出错: {str(e)}")
        return False

def main():
    page = None
    while True:
        try:
            print("请求新任务...")
            task = get_task()
            if not task:
                print("没有可用任务或获取任务失败。等待后重试...")
                time.sleep(60)
                continue
            print(f"收到任务，URL: {task['url']}")
            result, page = handle_website_crawling(task)
            success = submit_result(result)
            if not success:
                print("提交结果失败。等待后继续...")
                time.sleep(10)
            time.sleep(5)
        except Exception as e:
            print(f"主循环中出现意外错误: {str(e)}")
            if page:
                try:
                    page.quit()
                except:
                    pass
                page = None
            time.sleep(30)

if __name__ == "__main__":
    main()
