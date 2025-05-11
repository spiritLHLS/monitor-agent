import asyncio
import os
import json
import random
from datetime import datetime
from DrissionPage import ChromiumOptions, ChromiumPage
from app.const import DefaultUserAgent, IS_LINUX, IS_MAC
from app.servers import get_click_xy
from app.utils import check_path
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import uvicorn

# Mac
r = 2 if IS_MAC else 1
file_path = "images"
check_path(file_path)
file_suffix = "png"

class CookieRequest(BaseModel):
    url: str
    user_agent: str
    proxy_server: str = None

class Cloudflare5sScreenshotBypass:
    def __init__(self, user_agent=None, proxy_server=None):
        browser_path = "/usr/bin/google-chrome"
        options = ChromiumOptions()
        options.set_paths(browser_path=browser_path)
        user_agent = user_agent or DefaultUserAgent
        if user_agent:
            options.set_user_agent(user_agent)
        if proxy_server:
            print("proxy_server", proxy_server)
            options.set_proxy(proxy_server)
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
        if IS_LINUX:
            arguments.extend(["--start-maximized", "--no-sandbox"])
        else:
            arguments.append("--start-fullscreen")
        for argument in arguments:
            options.set_argument(argument)
        options.headless(False)
        self.driver = ChromiumPage(addr_or_opts=options)
    def random_sleep(self, base_seconds):
        """生成基准时间上下浮动1秒以内的随机等待时间"""
        return base_seconds + random.uniform(-0.5, 0.5)
    async def bypass(self):
        import pyautogui
        print(self.tag.cookies())
        screenshot = pyautogui.screenshot()
        file_name = datetime.now().strftime("%Y%m%d.%H_%M_%S")
        screenshot_path = f"{file_path}/{file_name}.{file_suffix}"
        screenshot.save(screenshot_path)
        for click_x, click_y in get_click_xy(screenshot_path):
            pyautogui.moveTo(click_x / r, click_y / r, duration=0.5, tween=pyautogui.easeInElastic)
            pyautogui.click()
            await asyncio.sleep(self.random_sleep(5))
        for line in self.tag.cookies():
            if line["name"] == "cf_clearance":
                return self.tag.cookies()
        return None
    async def get_cf_cookie(self, url, debug=False):
        self.driver.set.cookies.clear()
        tab_id = self.driver.new_tab(url).tab_id
        self.tag = self.driver.get_tab(tab_id)
        print("self.tag.rect.page_location", self.tag.rect.page_location)
        print(self.tag.user_agent)
        cookies = None
        # html_content = None
        await asyncio.sleep(self.random_sleep(5))
        for _ in range(15):
            print("Verification page detected. ", self.tag.title)
            cookies = await self.bypass()
            if cookies:
                await asyncio.sleep(self.random_sleep(6))
                # html_content = self.tag.html
                break
            await asyncio.sleep(self.random_sleep(3))
        self.tag.close()
        if not cookies:
            return None
        return {
            "user_agent": self.tag.user_agent,
            "cookies": cookies,
        }
    def __del__(self):
        if hasattr(self, 'driver'):
            self.driver.quit()

app = FastAPI()
@app.post("/get_cf_cookies")
async def get_cookies(request: CookieRequest):
    cf_bypass = Cloudflare5sScreenshotBypass(
        user_agent=request.user_agent,
        proxy_server=request.proxy_server
    )
    try:
        result = await cf_bypass.get_cf_cookie(request.url)
        if not result:
            raise HTTPException(status_code=400, detail="Failed to get CloudFlare cookies")
        return result
    finally:
        del cf_bypass

if __name__ == "__main__":
    uvicorn.run("app:app", host="127.0.0.1", port=8000, reload=True)