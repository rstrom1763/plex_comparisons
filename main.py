import json
import plex_compare as pc

conf_file = open("./config.json", 'r')
conf = json.load(conf_file)
conf_file.close()

url = "http://" + conf["plex_server_ip"] + ":32400"
pc.transcribe_data_csv(pc.get_library(
    url, conf["plex_token"], conf["library_name"]), out_file=conf["output_csv"])

print("File saved to " + conf["output_csv"])