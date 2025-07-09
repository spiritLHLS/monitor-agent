#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import time
import random
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

def get_task(flag: str) -> Optional[Dict]:
    try:
        url = f"{API_BASE_URL}{GET_TASK_ENDPOINT}?flag={flag}"
        payload = {"token": TOKEN}
        headers = {"Content-Type": "application/json"}
        response = requests.post(url, json=payload, headers=headers, timeout=6)
        response_data = response.json()
        if response.status_code == 200 and response_data.get("code") == 0:
            return response_data.get("data")
        else:
            print(f"Failed to get {flag} task: {response_data.get('msg')}")
            return None
    except Exception as e:
        print(f"Error while getting {flag} task: {str(e)}")
        return None

def handle_website_crawling(task: Dict, task_type: str) -> Dict:
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
            if "cf5s" in task_type:
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
            else:
                time.sleep(20)
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
        if response.status_code == 200 and response_data.get("code") == 0:
            print(f"Result submitted successfully: {response_data.get('msg')}")
            return True
        else:
            print(f"Failed to submit result: {response_data.get('msg')}")
            return False
    except Exception as e:
        print(f"Error while submitting result: {str(e)}")
        return False

def add_jitter(duration_seconds):
    jitter = random.uniform(0, duration_seconds * 0.5)
    return duration_seconds + jitter

def process_task_type(task_type: str, page):
    print(f"Requesting {task_type} task...")
    task = get_task(task_type)
    if not task:
        print(f"No available {task_type} task or failed to get task.")
        return page, False
    print(f"Received {task_type} task, URL: {task['url']}")
    result, page = handle_website_crawling(task, task_type)
    success = submit_result(result)
    if not success:
        print(f"Failed to submit {task_type} result.")
    return page, success

def main():
    page = None
    initial_backoff = 1
    max_backoff = 60
    while True:
        backoff = initial_backoff
        try:
            page, cf5s_success = process_task_type("cf5s", page)
            if cf5s_success:
                time.sleep(5)
            page, dynamic_success = process_task_type("dynamic", page)
            if dynamic_success:
                time.sleep(5)
            if not cf5s_success and not dynamic_success:
                print("Both task types failed. Waiting before retry...")
                time.sleep(60)
        except Exception as e:
            print(f"Unexpected error in main loop: {str(e)}")
            if page:
                try:
                    page.quit()
                except:
                    pass
                page = None
            wait = add_jitter(backoff)
            print(f"Error encountered. Waiting {wait}s before retrying...")
            time.sleep(wait)
            backoff = min(max_backoff, backoff * 2)

if __name__ == "__main__":
    main()
