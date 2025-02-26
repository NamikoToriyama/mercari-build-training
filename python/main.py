import os
import logging
import pathlib
import json
import hashlib
from fastapi import FastAPI, Form, HTTPException, File, UploadFile
from fastapi.responses import FileResponse
from fastapi.middleware.cors import CORSMiddleware

app = FastAPI()
logger = logging.getLogger("uvicorn")
logger.level = logging.INFO
images = pathlib.Path(__file__).parent.resolve() / "images"
origins = [os.environ.get("FRONT_URL", "http://localhost:3000")]
app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=False,
    allow_methods=["GET", "POST", "PUT", "DELETE"],
    allow_headers=["*"],
)

JSON_PATH = '../db/items.json'
IMAGE_DIR = '../images/'

def load_json_to_dict():
    with open(JSON_PATH, 'r', encoding='utf-8') as file:
        data = json.load(file)
    return data

def write_dict_to_json(data_dict):
    with open(JSON_PATH, 'w', encoding='utf-8') as json_file:
        json.dump(data_dict, json_file, ensure_ascii=False, indent=4)


def hash_string_sha256(input_string):
    sha256_hash = hashlib.sha256()
    sha256_hash.update(input_string.encode('utf-8'))
    return sha256_hash.hexdigest()

def save_image(image: UploadFile):
   filename = f"{hash_string_sha256(image.filename)}.png"
   with open(f"{images}/{filename}" , "wb") as buffer:
        buffer.write(image.file.read())
   return filename

@app.get("/")
def root():
    return {"message": "Hello, world!"}


@app.post("/items")
def add_item(name: str = Form(...), category: str = Form(...), image: UploadFile = File(...)):
    logger.info(f"Receive item: {name}")
    hashed_name = save_image(image)
    data = load_json_to_dict()['items']
    data.append({"name": name, "category": category, "image": hashed_name})
    write_dict_to_json(data)
    return {"message": f"item received: {name}"}

@app.get("/items")
def list_items():
    logger.info(f"list items")
    data = load_json_to_dict()
    return {"items": data}

@app.get("/items/{item_id}")
def get_item(item_id):
    logger.info(f"get item")
    id = int(item_id)-1
    try:
        item = load_json_to_dict()[id]
    except Exception as e:
        raise ValueError(f"item not found item_id = {id}; error = {e}")
    return item

@app.get("/image/{image_name}")
async def get_image(image_name):
    # Create image path
    image = images / image_name

    if not image_name.endswith(".jpg"):
        raise HTTPException(status_code=400, detail="Image path does not end with .jpg")

    if not image.exists():
        logger.info(f"Image not found: {image}")
        image = images / "default.jpg"

    return FileResponse(image)
