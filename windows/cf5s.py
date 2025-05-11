import random
import time
import cv2
import pytesseract
import pyautogui
import os
from datetime import datetime

pytesseract.pytesseract.tesseract_cmd = 'C:/Program Files/Tesseract-OCR/tesseract.exe'
pyautogui.PAUSE = 1

# https://github.com/UB-Mannheim/tesseract/wiki need to install tesseract-ocr first
def get_click_xy(image_path):
    image = cv2.imread(image_path)
    gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
    edges = cv2.Canny(gray, 50, 150)
    contours, _ = cv2.findContours(edges, cv2.RETR_TREE, cv2.CHAIN_APPROX_SIMPLE)
    min_area = 10000
    max_area = 100000
    contour_boxes = []
    small_boxes = []
    for contour in contours:
        area = cv2.contourArea(contour)
        if min_area < area < max_area:
            x, y, w, h = cv2.boundingRect(contour)
            roi = gray[y:y + h, x:x + w]
            text = pytesseract.image_to_string(roi)
            if "verify you are human" in text.lower():
                contour_boxes.append((x, y, w, h))
                roi_edges = cv2.Canny(roi, 50, 150)
                small_contours, _ = cv2.findContours(roi_edges, cv2.RETR_TREE, cv2.CHAIN_APPROX_SIMPLE)
                for small_contour in small_contours:
                    small_area = cv2.contourArea(small_contour)
                    if 500 < small_area < 5000:
                        sx, sy, sw, sh = cv2.boundingRect(small_contour)
                        small_boxes.append((x + sx, y + sy, sw, sh))
    for (x, y, w, h) in contour_boxes + small_boxes:
        cv2.rectangle(image, (x, y), (x + w, y + h), (0, 255, 0), 2)
    click_xy = set()
    if small_boxes:
        for (x, y, w, h) in small_boxes:
            click_x = x + w // 2
            click_y = y + h // 2
            cv2.circle(image, (click_x, click_y), 5, (0, 0, 255), -1)
            click_xy.add((click_x, click_y))
    else:
        for (x, y, w, h) in contour_boxes:
            click_x = x + w // 2
            click_y = y + h // 2
            cv2.circle(image, (click_x, click_y), 5, (0, 0, 255), -1)
            click_xy.add((click_x, click_y))
    # cv2.imwrite(image_path + ".click.png", image)
    return click_xy

def detect_cf5s(time_wait=6):
    screenshots_dir = "screenshots"
    os.makedirs(screenshots_dir, exist_ok=True)
    file_name = datetime.now().strftime("%Y%m%d.%H_%M_%S") + ".png"
    screenshot_path = os.path.join(screenshots_dir, file_name)
    time.sleep(time_wait)
    try:
        screenshot = pyautogui.screenshot()
        screenshot.save(screenshot_path)
        click_positions = get_click_xy(screenshot_path)
        os.remove(screenshot_path)
        return len(click_positions) > 0
    except Exception as e:
        print(f"Detect ERROR: {e}")
        try:
            os.remove(screenshot_path)
        except:
            pass
        return False

def detect_success_verification(image_path):
    import cv2
    import numpy as np
    import pytesseract
    image = cv2.imread(image_path)
    gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
    edges = cv2.Canny(gray, 50, 150)
    contours, _ = cv2.findContours(edges, cv2.RETR_TREE, cv2.CHAIN_APPROX_SIMPLE)
    min_area = 1000
    max_area = 50000
    success_found = False
    for contour in contours:
        area = cv2.contourArea(contour)
        if min_area < area < max_area:
            x, y, w, h = cv2.boundingRect(contour)
            roi = gray[y:y + h, x:x + w]
            roi_edges = cv2.Canny(roi, 50, 150)
            circles = cv2.HoughCircles(
                roi_edges,
                cv2.HOUGH_GRADIENT,
                dp=1,
                minDist=20,
                param1=50,
                param2=30,
                minRadius=5,
                maxRadius=30
            )
            if circles is not None:
                circles = np.uint16(np.around(circles))
                for i in circles[0, :]:
                    center_x, center_y = i[0], i[1]
                    radius = i[2]
                    global_center_x = x + center_x
                    global_center_y = y + center_y
                    #cv2.circle(image, (global_center_x, global_center_y), radius, (0, 255, 0), 2)
                    #cv2.circle(image, (global_center_x, global_center_y), 2, (0, 0, 255), 3)
                    success_found = True
                    break
            if success_found:
                break
    # cv2.imwrite(f"{image_path}.debug.png", image)
    return success_found

def pass_cf5s(page):
    tag = page.get_tabs()[-1]
    if not detect_cf5s():
        return True
    screenshots_dir = "screenshots"
    os.makedirs(screenshots_dir, exist_ok=True)
    for _ in range(5):
        time.sleep(2)
        for _ in range(5):
            screenshot_path = os.path.join(screenshots_dir, "cf_screenshot.png")
            pyautogui.screenshot().save(screenshot_path)
            click_positions = get_click_xy(screenshot_path)
            os.remove(screenshot_path)
            if not click_positions:
                continue
            for (x, y) in click_positions:
                pyautogui.moveTo(x, y, duration=0)
                pyautogui.click()
                time.sleep(random.uniform(2, 3))
                safe_x = max(0, x - 100)
                safe_y = max(0, y - 100)
                pyautogui.moveTo(safe_x, safe_y, duration=0)
                time.sleep(random.uniform(0, 1))
                pyautogui.screenshot().save(screenshot_path)
                success = detect_success_verification(screenshot_path)
                os.remove(screenshot_path)
                if success:
                    print("PASS!")
                    return True
        tag.refresh()
        time.sleep(4)
    return False
