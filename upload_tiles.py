import os
import glob
import re
from instagrapi import Client
from instagrapi.exceptions import LoginRequired
import logging
import time
from dotenv import load_dotenv # Import the library

# Load environment variables from .env file
load_dotenv()

# --- Configuration ---
TILES_DIR = "tiles"
INSTAGRAM_USERNAME = os.environ.get("INSTA_USER") # Get from environment variable
INSTAGRAM_PASSWORD = os.environ.get("INSTA_PASS") # Get from environment variable
# Optional: Specify session file path to reuse login session
SESSION_FILE = "session.json"
# Optional: Caption for the posts (can be customized per post if needed)
CAPTION = """
oyeoyepuedequeestoestéfuncionando
"""
UPLOAD_DELAY_SECONDS = 5 # Add a small delay between uploads

# Setup basic logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

def get_sorted_tiles(directory):
    """Finds and sorts tile images numerically for correct Instagram upload order (1.jpg first)."""
    pattern = os.path.join(directory, "tile_*.jpg")
    files = glob.glob(pattern)
    logging.debug(f"Found files before sort: {[os.path.basename(f) for f in files]}")

    def sort_key(filepath):
        filename = os.path.basename(filepath)
        # Match one or more digits between tile_ and .jpg
        match = re.match(r"^tile_(\\d+)\\.jpg$", filename)
        if match:
            num = int(match.group(1))

            if num <= 0: # Tile numbers should be positive.
                logging.warning(f"File {filename} has non-positive number {num}. Will be sorted last.")
                return (float('inf'), float('inf'))

            # Assuming a fixed number of 3 columns for the grid, as specified.
            # The number of rows can vary.
            # This key ensures sorting happens in the natural reading order:
            # tile_1, tile_2, tile_3 (first row)
            # tile_4, tile_5, tile_6 (second row)
            # and so on.
            fixed_cols = 3
            r = (num - 1) // fixed_cols
            c = (num - 1) % fixed_cols
            logging.debug(f"File: {filename}, Num: {num}, Key: {(r, c)}")
            return (r, c) # Sort by (row, col) tuple
        else:
            logging.warning(f"Filename {filename} does not match expected pattern '^tile_\\\\d+\\\\.jpg$'. Will be sorted last.")
            # Sort unmatchable files last
            return (float('inf'), float('inf'))

    # Sort files based on the (row, col) key, ascending order (default).
    # This should result in ['tile_1.jpg', 'tile_2.jpg', ..., 'tile_9.jpg'] for a standard 3x3 grid.
    try:
        # Sort ascending (reverse=False is the default, making it explicit)
        files.sort(key=sort_key, reverse=False)
    except Exception as e:
        logging.error(f"Error during sorting: {e}")
        # Return unsorted or partially sorted list in case of error
        return files

    if not files:
        logging.warning(f"No files matching '{pattern}' found in '{directory}'")
    else:
        # Log the order *after* sorting to verify
        logging.info(f"Found {len(files)} tiles. Sorted upload order (should be tile_1.jpg first):")
        sorted_basenames = [os.path.basename(f) for f in files]
        # Log the full sorted list for clarity
        logging.info(f"  - Files: {sorted_basenames}")

    return files

def login_client():
    """Creates, logs in, and returns an instagrapi Client instance."""
    cl = Client()

    # Load session if exists
    if os.path.exists(SESSION_FILE):
        try:
            cl.load_settings(SESSION_FILE)
            logging.info(f"Loaded session from {SESSION_FILE}")
            cl.login(INSTAGRAM_USERNAME, INSTAGRAM_PASSWORD)
            # Check if session is valid
            try:
                 cl.get_timeline_feed()
                 logging.info("Session is valid.")
            except LoginRequired:
                 logging.warning("Session expired or invalid. Re-login required.")
                 os.remove(SESSION_FILE) # Remove invalid session file
                 cl = Client() # Re-initialize client
                 cl.login(INSTAGRAM_USERNAME, INSTAGRAM_PASSWORD)
        except Exception as e:
            logging.warning(f"Could not load session: {e}. Performing full login.")
            cl = Client()
            cl.login(INSTAGRAM_USERNAME, INSTAGRAM_PASSWORD)
    else:
        logging.info("No session file found. Performing full login.")
        cl.login(INSTAGRAM_USERNAME, INSTAGRAM_PASSWORD)

    # Save session for future use
    cl.dump_settings(SESSION_FILE)
    logging.info(f"Session saved to {SESSION_FILE}")
    return cl

def main():
    if not INSTAGRAM_USERNAME or not INSTAGRAM_PASSWORD:
        logging.error("Error: INSTA_USER and INSTA_PASS environment variables must be set.")
        return

    tiles_to_upload = get_sorted_tiles(TILES_DIR)
    if not tiles_to_upload:
        logging.info("No tiles found to upload.")
        return

    try:
        client = login_client()
        logging.info(f"Logged in as {INSTAGRAM_USERNAME}")
    except LoginRequired:
        logging.error("Login failed: LoginRequired exception. Check credentials or 2FA settings.")
        return
    except Exception as e:
        logging.error(f"Login failed: {e}")
        return

    logging.info("Starting upload process...")
    successful_uploads = 0
    for i, tile_path in enumerate(tiles_to_upload):
        logging.info(f"Uploading tile {i+1}/{len(tiles_to_upload)}: {os.path.basename(tile_path)}...")
        try:
            media = client.photo_upload(tile_path, caption=CAPTION)
            logging.info(f"  Successfully uploaded: {media.pk} (Code: {media.code})")
            successful_uploads += 1
            # Add a delay to avoid rate limiting
            if i < len(tiles_to_upload) - 1:
                 logging.info(f"  Waiting {UPLOAD_DELAY_SECONDS} seconds...")
                 time.sleep(UPLOAD_DELAY_SECONDS)
        except Exception as e:
            logging.error(f"  Failed to upload {os.path.basename(tile_path)}: {e}")
            # Optional: Decide whether to stop or continue on error
            # break # Uncomment to stop after the first error

    logging.info(f"Upload process finished. {successful_uploads}/{len(tiles_to_upload)} tiles uploaded successfully.")

if __name__ == "__main__":
    main() 