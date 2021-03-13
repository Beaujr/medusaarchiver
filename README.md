# MedusaArchiver

This is a quick binary written to interact with the Medusa API to automatically change statuses of files based on external logic.
for example to change all **Downloaded** Episodes to **Archived**

```yaml
medusaarchiver \
-medusaUrl=http://127.0.0.1:8081 \
-token="medusa api key" \
-target=6 \
-current=4
```


# Target and Current Flags
Pass the value of the Id columns as the flags ``-target=`` and ``-current=`` The Value column is the corresponding status

| Id | Value |
|--|--|
|-1|Unset|
|1|Unaired|
|2|Snatched|
|3|Wanted|
|4|Downloaded|
|5|Skipped|
|6|Archived|
|7|Ignored|
|9|Snatched (Proper)|
|10|Subtitled|
|11|Failed|
|12|Snatched (Best)|