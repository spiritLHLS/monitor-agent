#!/usr/bin/env python3
# -*- coding: utf-8 -*-
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
        response = requests.post(url, json=payload, headers=headers, timeout=6)
        response_data = response.json()
        if response.status_code == 200 and response_data.get("success"):
            return response_data.get("data")
        else:
            print(f"Failed to get task: {response_data.get('msg')}")
            return None
    except Exception as e:
        print(f"Error while getting task: {str(e)}")
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
            print(f"Attempting to load ({attempt + 1}/{MAX_ATTEMPTS}): {task['url']}")
            task_start_time = time.time()
            page.latest_tab.get(task["url"], timeout=MAX_TASK_RUNTIME)
            time.sleep(15)
            if detect_cf5s():
                print("Detected cf5s protection, trying to bypass...")
                if not pass_cf5s(page):
                    print("Failed to bypass cf5s")
                    raise Exception("Failed to bypass cf5s")
                else:
                    page.latest_tab.get(task["url"], timeout=MAX_TASK_RUNTIME)
                    print("Successfully bypassed cf5s")
                    time.sleep(6)
            if time.time() - task_start_time > MAX_TASK_RUNTIME:
                raise Exception("Task execution timeout")
            page_source = page.html
            result["success"] = True
            result["webdata"] = page_source
            break
        except Exception as e:
            error_msg = f"Error during crawling (attempt {attempt + 1}/{MAX_ATTEMPTS}): {str(e)}"
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
                result["error_msg"] = f"Task failed after {MAX_ATTEMPTS} attempts: {str(e)}"
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
            print(f"Result submitted successfully: {response_data.get('msg')}")
            return True
        else:
            print(f"Failed to submit result: {response_data.get('msg')}")
            return False
    except Exception as e:
        print(f"Error while submitting result: {str(e)}")
        return False

def main():
    page = None
    while True:
        try:
            print("Requesting new task...")
            task = get_task()
            if not task:
                print("No available task or failed to get task. Retrying later...")
                time.sleep(60)
                continue
            print(f"Received task, URL: {task['url']}")
            result, page = handle_website_crawling(task)
            success = submit_result(result)
            if not success:
                print("Failed to submit result. Waiting before continuing...")
                time.sleep(10)
            time.sleep(5)
        except Exception as e:
            print(f"Unexpected error in main loop: {str(e)}")
            if page:
                try:
                    page.quit()
                except:
                    pass
                page = None
            time.sleep(30)

if __name__ == "__main__":
    main()
