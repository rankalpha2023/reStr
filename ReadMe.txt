reStr: batch replace string

Usage of reStr:
  --dir , -d
        string: Root directory to search (default ".")
  --from, -f
        string: String to search for (case-sensitive)
  --to, -t
        string: String to replace with
  --verbose, -v
        bool: Verbose output
  --workers, -w
        int: Number of worker goroutines (default 4)
  --test, -T
        bool: Dry run without actually replacement

example:
  reStr -f "frida" -t "panda" -T -v -d /mnt/workspace/frida/frida-patch