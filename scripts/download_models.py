# Copyright (c) 2026 Michael Lechner
# Licensed under the MIT License. See LICENSE file in the project root for full license information.

import yaml
import os
from huggingface_hub import hf_hub_download

def download():
    with open("config.yaml", "r") as f:
        config = yaml.safe_load(f)

    cache_dir = config["storage"]["cache_dir"]
    models = config["models"]["available"]
    token = os.getenv("HF_TOKEN")

    # Mappings für FastEmbed ONNX Repos
    mappings = {
        "multilingual-e5-small": {
            "repo": "Qdrant/multilingual-e5-small-onnx",
            "files": ["onnx/model.onnx", "tokenizer.json", "config.json", "tokenizer_config.json"]
        },
        "BAAI/bge-small-en-v1.5": {
            "repo": "qdrant/bge-small-en-v1.5-onnx-q",
            "files": ["model.onnx", "tokenizer.json", "config.json", "tokenizer_config.json"]
        }
    }

    for m in models:
        name = m["name"]
        if name not in mappings:
            print(f"Skipping {name}, no mapping found.")
            continue
        
        print(f"\n--- Downloading Model: {name} ---")
        mapping = mappings[name]
        
        # FastEmbed Ordnerstruktur simulieren
        folder_name = f"models--{name.replace('/', '--')}"
        if "/" not in name:
            folder_name = f"models--qdrant--{name}"
            
        local_dir = os.path.join(cache_dir, folder_name)
        
        for file in mapping["files"]:
            print(f"Fetching {file}...")
            try:
                hf_hub_download(
                    repo_id=mapping["repo"],
                    filename=file,
                    local_dir=local_dir,
                    local_dir_use_symlinks=False,
                    token=token
                )
            except Exception as e:
                print(f"Error downloading {file}: {e}")

if __name__ == "__main__":
    download()
