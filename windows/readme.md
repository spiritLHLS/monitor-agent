python3.10.11

1. Open **Control Panel**

2. Go to **System and Security** → **System**

3. Click **Advanced system settings** (on the left sidebar)

4. In the popup, click **Environment Variables...**

5. In the **System variables** section, scroll to find the variable named `Path`, then click **Edit...**

6. Click **New**, then paste the Tesseract path, e.g.:

   ```
   C:\Program Files\Tesseract-OCR
   ```

7. Click **OK** on all windows to apply changes.


   ```cmd
   tesseract -v
   ```
