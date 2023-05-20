import json
import plex_compare as pc
import os
from platform import system

system_os = system() # Get the type of system this script is running on. (Windows or Linux)

def clear(): # Clear the terminal
    if system_os == "Windows": # Use the Windows cls command if Windows machine
        os.system('cls') # cls works both in powershell and cmd
    else:
        os.system('clear')

menu = \
"""\n1. Query Plex server
2. Compare libraries
3. Reset config file
4. Exit\n
Please select an option: """

clear()
while True:
    conf_file = open("./config.json", 'r') # Opens the conf file for reading
    conf = json.load(conf_file) # Load conf_file contents as json to the var conf
    conf_file.close() # Close the conf file

    # Make sure that every field in the conf file is filled out
    # If a field is not filled out, prompt the user for the value. 
    for item in conf:
        if conf[item] == "":
            conf[item] = input("Please input config value for " + item + ": ")
            open("./config.json",'w').write(json.dumps(conf))
            clear()

    choice = int(input(menu)) # Prompts user for their menu choice
    
    if choice == 1: # Query Plex Server

        # Ask user for the various parameters needed to query the server
        clear()
        out_file = input("Please enter path for the output csv file: ")
        library_name = input("Please enter the name of the library to query: ")
        url = "http://" + conf["plex_server_ip"] + ":32400"

        print("\nWorking...\n") # To tell the user that the script is doing its thing
        library = pc.get_library(url, conf["plex_token"], library_name) # Query server for the data
        pc.transcribe_data_csv(library, out_file=out_file) # Write the data to a csv file

        clear()
        print("File saved to " + out_file)
    
    elif choice == 2: # Compare libraries

        # Print out some basic instructions
        clear()
        instructions = """-Finds entries that are in the first CSV but not in the second
        -Inputs are file paths to CSV files exported with this script using the \"Query Plex Server\" option
        -Output is another CSV file
        -Make sure the CSVs to be compared are of the same media type(Movie or Show)"""
        print(instructions)

        # Prompt user for the needed parameters
        db1 = input("Please enter path to CSV1: ")
        db2 = input("Please enter path to CSV2: ")
        output_file = input("Please enter path for an output CSV file: ")
        exclude_file = input("Would you like to use an exclude file? Y or N: ").lower()

        # If user wants to use an exclude file prompt them for it and compare using it
        # If they don't want to use it, compare without exclude list
        # If they put in an invalid option, notify them and return to menu
        if exclude_file == 'y':
            exclude_file = input("Please enter path to the exclude file: ")
            clear()
            pc.plex_compare(db1, db2, output_file, exclude_file)
        elif exclude_file == "n":
            clear()
            pc.plex_compare(db1, db2, output_file)
        else:
            clear()
            print("Invalid option!\n\n")
            continue
    
    elif choice == 3: # Reset config file

        # Loop throught the config dict, set all values to an empty string
        # Then overwrite the config file with the json dump of the conf dict
        for item in conf:
            conf[item] = ""
        open("./config.json",'w').write(json.dumps(conf))
        clear()
        print("Config file has been reset!\n\n")
    
    elif choice == 4: # Exit the script
        clear()
        print("Exiting...\n\n")
        exit()

    else: # Restarts the loop if the choice was not a valid option
        clear()
        print("Invalid option!\n\n")
        continue