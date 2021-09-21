def import_csv(csv):

    import re

    csv = csv.replace("\\", "/")  # Ensure no errors due to wrong slash
    # Opens the CSV for reading
    csv = (open(csv, 'r', encoding="utf-8")).readlines()
    headers = csv[0].split(",")  # Seperates headers by comma into a list
    csv.remove(csv[0])  # Deletes headers row from csv data
    list = []

    # Get rid of quotations in headers
    for header in headers:
        headers[headers.index(header)] = header.replace('"', "")

    # Creates dictionaries from CSV lines and appends them to a list
    for line in csv:
        dict = {}
        if line[0] == '"':
            line = re.split(r',(?=")', line)
        else:
            line = line.split(',')

        for header in headers:
            dict[header] = line[headers.index(header)].replace('"', "")

        list.append(dict)

    # Output is a list of dictionaries crafted from the CSV lines
    return(list)


def export_csv(dict_list, file):

    file = file.replace("\\", "/")  # Ensure no errors due to wrong slash
    csv_header = ""
    if not dict_list:
        print("List is empty!")
        return
    headers = dict_list[0].keys()  # Gets a list of all headers
    # Clears content of output file if existing else creates it
    open(file, 'w').close()
    file = open(file, 'a', encoding="utf-8")  # Opens output file for appending

    # Creates line 1 of CSV with all of the headers
    for header in headers:
        csv_header = csv_header + '"' + header.replace("\n", "") + '",'
    file.write(csv_header + "\n")

    # Converts the dictionaries into csv lines and appends to file
    for dict in dict_list:
        for header in headers:
            file.write('"' + dict[header].replace("\n", "") + '",')
        file.write("\n")

    file.close  # Close output file to free the memory

def human_readable(num, suffix='B'):
    # Returns inputted byte count as human readable text

    for unit in ['', 'Ki', 'Mi', 'Gi', 'Ti', 'Pi', 'Ei', 'Zi']:
        if abs(num) < 1024.0:
            return "%3.1f%s%s" % (num, unit, suffix)
        num /= 1024.0

    return "%.1f%s%s" % (num, 'Yi', suffix)
