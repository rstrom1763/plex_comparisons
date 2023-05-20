# Ryan Strom

# exports as a CSV what movies are in the first DB and not in the second
# inputs are the file paths to each of the CSV files.

def plex_compare(db1, db2, file, exclude_file=""):

    import utils
    import json

    space_needed = 0 # Variable to hold the number of bytes difference between the two libraries

    # Load the exclude file into a list and remove newline characters
    if exclude_file != "":
        exclude_file = (open(exclude_file, 'r', encoding='utf-8')).readlines()
        for line in exclude_file:
            exclude_file[exclude_file.index(line)] = line.replace("\n", "")

    # Import both DB files into list of dictionaries
    if ".csv" in db1:
        db1 = utils.import_csv(db1)
    elif ".json" in db1:
        db1 = json.load(db1)
    else:
        print("Unknown data type")
        return
    if ".csv" in db2:
        db2 = utils.import_csv(db2)
    elif ".json" in db2:
        db2 = json.load(db2)
    else:
        print("Unknown data type")
        return

    dict1 = {} # Dict that db1 will be loaded into
    dict2 = {} # Dict that db2 will be loaded into
    list = [] # List of media items that are found in the first db but not the second. For final output

    # Load the db data into the dicts
    # Title of the media item becomes the key
    # The whole individual dict becomes the value
    for dict in db1:
        dict1[dict["Title"].lower()] = dict
    for dict in db2:
        dict2[dict["Title"].lower()] = dict

    # Perform the compare
    # Loops through all of the media items in dict1 and adds andything not also in dict 2 to "list"
    # Also needs to not be in the exclude file
    for movie in dict1:
        if movie not in dict2 and movie not in exclude_file:
            list.append(dict1[movie])

# Iterates through the dictionaries to print required file space
    for item in list:
        if item["FileSizeBytes"].isdecimal():
            space_needed += int(item["FileSizeBytes"])
    print(utils.human_readable(space_needed))

    utils.export_csv(list, file) # Writes the resulting data to a csv file

# Queries the Plex Server for the library data
# Url is the dns name or the ip address of the plex server you are querying
# Token is the Plex server token. This is taken from the web when you view the xml for a media item. It will be at the very end
# Library is a the name of the library you are wishing to query the data for
def get_library(url, token, library):

    from plexapi.server import PlexServer

    # print(type(test[0].locations))
    results = ((PlexServer(url, token)).library.section(library)).search()

    # If the library is TV Shows, return a list of episode objects
    if hasattr(results[0], 'season'):
        episode_list = []
        for show in results:
            for season in show.seasons():
                for episode in season.episodes():
                    episode.title = "{} : {}".format(show.title, episode.title)
                    episode_list += episode
        return episode_list

    # Else return the movie file list
    return results


def transcribe_data_csv(files, out_file):

    import utils

    out_file = open(out_file, 'w', encoding="utf-8")

    out_file.write('"{}","{}","{}","{}","{}","{}","{}","{}","{}"\n'.format(
        "Title", "Year", "FileSize", "FileSizeBytes", "bitrate", "resolution", "codec", "container", "FilePath"))

    for file in files:
        size = file.media[0].parts[0].size
        bitrate = file.media[0].bitrate
        resolution = file.media[0].videoResolution
        container = file.media[0].parts[0].container
        videoCodec = file.media[0].videoCodec
        out_file.write('"{}","{}","{}","{}","{}","{}","{}","{}","{}"\n'.format(
            file.title, file.year, utils.human_readable(size), size, bitrate, resolution, videoCodec, container, file.locations[0]))

    out_file.close()


def transcribe_data_json(files):

    import utils
    import json

    dict = {}

    for file in files:

        size = file.media[0].parts[0].size
        file_size_human = utils.human_readable(size)
        bitrate = file.media[0].bitrate
        resolution = file.media[0].videoResolution
        container = file.media[0].parts[0].container
        videoCodec = file.media[0].videoCodec
        dict[file.title] = {"title": file.title, "year": file.year,
                            "file_size_human_readable": file_size_human, "file_size_bytes": size, "bitrate": bitrate, "resolution": resolution,  "codec": videoCodec, "container": container, "filepath": file.locations[0]}

    return json.dumps(dict)


def sync_data(json, url, token):

    import requests

    headers_dict = {"token": token, 'Content-type': 'application/json', 'Accept': 'text/plain'}
    requests.post(url, headers=headers_dict, data=json, verify=False)


def download_diff(url, token, out_file):
    # Functional: Sept 19 2021

    import requests

    headers_dict = {"token": token}

    out_file = open(out_file, 'w', encoding='utf-8')

    out_file.write(requests.get(url, headers=headers_dict, verify=False).json())

    out_file.close()


def new_user(username, url, password):
    import requests
    import string
    import random
    import json

    N = 20
    token = ''.join(random.SystemRandom().choice(string.ascii_letters + string.digits) for _ in range(N))

    headers_dict = {'Content-type': 'application/json', 'Accept': 'text/plain', 'access_token': token}

    data_json = {'username': username,'access_token': token, 'password': password}

    data_json = json.dumps(data_json)
    result = requests.post(url, headers=headers_dict, data=data_json, verify=False)

    if result.status_code != 200:
        print("Oh no")
