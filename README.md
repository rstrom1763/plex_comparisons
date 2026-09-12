# 📦 Plex Compare

A command-line tool to **export** and **compare** Plex
library content.


--------------------

## 📄 Table of Contents

-   [Getting Library Dumps](#getting-library-dumps)
-   [Usage: Dumping](#usage-dumping)
-   [Usage: Comparing](#usage-comparing)
-   [Supported Media Types](#supported-media-types)
-   [Example](#example)

------------------------------------------------------------------------
<a id="getting-library-dumps"></a>
# 📥 Getting Library Dumps

## 1️⃣ Configure Your Environment

Set the `PLEX_DB_PATH` variable inside your `.env` file:

    PLEX_DB_PATH="/var/lib/plexmediaserver/Library/Application Support/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db"

> 💡 This is the default Plex database path on Ubuntu installations. Set it to wherever  your plex db file is

For Plex running in a container, optionally map movie, episode, and song paths
to the host OS in `.env`. All media types share the same mapping using
`PLEX_PATH_OVERRIDE_FROM` and `PLEX_PATH_OVERRIDE_TO`:

```dotenv
PLEX_PATH_OVERRIDE_FROM="/media/"
PLEX_PATH_OVERRIDE_TO="/mnt/media/"
```

These settings turn `/media/movies/Alien.mkv` into `/mnt/media/movies/Alien.mkv`,
`/media/tv/Show/S01E01.mkv` into `/mnt/media/tv/Show/S01E01.mkv`, and
`/media/music/Artist/Song.flac` into `/mnt/media/music/Artist/Song.flac` when
loading media from Plex, so exported CSVs contain the corrected paths.
Matching is a literal, case-sensitive prefix replacement. Include trailing
separators as shown to match a directory precisely; separators are not converted.
If either value is empty or the prefix does not match, the path is unchanged.
CSV imports are not remapped.

------------------------------------------------------------------------
<a id="usage-dumping"></a>
# 🔧 Usage: Dumping

Create CSV dumps of your Plex library:

``` bash
./plex_comparisons dump
```

This command generates the following files in the current directory:

-   `movies.csv`
-   `episodes.csv`
-   `songs.csv`

These CSV files contain your Plex library data and are used for
comparisons.

------------------------------------------------------------------------
<a id="usage-comparing"></a>
# 🔍 Usage: Comparing

Compare two library dumps to see what's missing from each.

## 📌 Syntax

``` bash
./plex_comparisons compare <file1.csv> <file2.csv> <media-type>
```

Where:

-   `<file1.csv>` is your first exported library dump
-   `<file2.csv>` is your second exported library dump
-   `<media-type>` specifies the type of media in both files

------------------------------------------------------------------------
<a id="supported-media-types"></a>
# 🎞 Supported Media Types
Use one of the following:

-   `movie`
-   `show`
-   `song`

> ⚠️ **Important:**\
> Both CSV files must be for the same media type.

------------------------------------------------------------------------
<a id="example"></a>
# 📋 Example
Compare two movie library dumps:

``` bash
./plex_comparisons compare ./movieLibraryDump1.csv ./movieLibraryDump2.csv movie
```

This will generate:

-   `file1_no_have.csv`
-   `file2_no_have.csv`

These contain the media items that exist in one dump but not the other.

------------------------------------------------------------------------
