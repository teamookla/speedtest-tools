## speedtest-extract
This tool interacts with the Speedtest Intelligence Data Extracts API allowing you to list and download your files. 

### Usage

```
NAME:
   speedtest-extract - Download extract files for Speedtest Intelligence

USAGE:
   speedtest-extract [global options] command [command options] [arguments...]

COMMANDS:
   list      List available extracts
   download  Download extract files
   help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --all                     Show all extract files, not just latest available (default: false)
   --config value            Specify the config file (default: "speedtest-extract.yaml")
   --filter-filenames value  Limit extracts to this comma-delimited list of filenames
   --filter-groups value     Limit extracts to this comma-delimited list of groups
   --since value             Limit extracts to ones updated since the provided date (YYYY-MM-DD)
   --verbose                 Enable verbose logging to help with debugging (default: false)
   --help, -h                show help (default: false)
   --version, -v             print the version (default: false)
```

### Examples

Note: A default config file will be generated for you on first run. Update it with your api key and secret.

#### List
* Show latest versions of extracts
```
speedtest-extract list
```
* Filter by specific extract groups, which are defined by the extract hierarchy directory structure.
```
speedtest-extract --filter-groups web,native list
```

* Show all extracts updated since a specific date
```
speedtest-extract --since 2022-01-01 list
```

#### Download

For all examples above, you can replace `list` with `download` to retrieve the files instead of listing them.

Files will download to the current directory, this can be changed by editing the `storage_directory` value in the config file. 
Additionally, to download the files into a hierarchy based on the group and dataset names (vs a flat list), use the `--use-file-hierarchy` flag.

By default, download will report the number of files and prompt to continue, abort, or list the files. 
The `--confirm` flag will skip this prompt and start downloading immediately.

If a file already exists, it will be skipped unless the `--overwrite-existing` flag is used.

### Request Caching

When enabled, request caching will speed up interactive filtering and viewing of the extracts list by caching the requests locally for a period of time. 
This is disabled by default as it has little use when executing the command periodically or unattended (via cron/etc), but can be helpful when searching through the extract list and formatting a command to retrieve specific files. 

To enable, set the `cache_duration_minutes` value in the config file to a positive integer and adjust the `cache_filename` if desired.

### Switching from the legacy python script

To replicate the functionality of the python script, use this command:
```
speedtest-extract download --confirm
```
