Prerequisites:
- Have go installed on your system
- Program must be run from your plex system(for now)

How to build this project:
 - Clone it onto your system
 - cd into the project
 - run ```go build .```
This will result in an executable binary file

How to run this project:
- On your system running plex, run ```./<path to executable>```
This will pull the data from your plex db and result in CSV file dumps of metadata of the various media types in the current directory.

Default path to plex DB is ```/var/lib/plexmediaserver/Library/Application Support/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db```
This value is hard coded for now, will implement ways to override this in the future
