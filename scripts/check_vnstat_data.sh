#!/bin/bash

echo "=== vnstat Data Export Verification ==="
echo "Date: $(date)"
echo ""

# 1. Check how many days of daily data are in JSON export
echo "=== Daily data count in JSON export ==="
vnstat --json a | jq '.interfaces[0].traffic.day | length'

# 2. Show the date range of daily data in JSON
echo -e "\n=== Date range of daily data in JSON ==="
echo "First day:"
vnstat --json a | jq '.interfaces[0].traffic.day[0].date'
echo "Last day:"
vnstat --json a | jq '.interfaces[0].traffic.day[-1].date'

# 3. Calculate total from daily data in JSON
echo -e "\n=== Total from daily data in JSON ==="
vnstat --json a | jq '.interfaces[0].traffic.day | map(.rx + .tx) | add / 1099511627776' | awk '{printf "%.2f TiB\n", $1}'

# 4. Check monthly data count in JSON
echo -e "\n=== Monthly data in JSON export ==="
echo "Count:"
vnstat --json a | jq '.interfaces[0].traffic.month | length'
echo "Data:"
vnstat --json a | jq '.interfaces[0].traffic.month'

# 5. Check yearly data in JSON
echo -e "\n=== Yearly data in JSON export ==="
echo "Count:"
vnstat --json a | jq '.interfaces[0].traffic.year | length'
echo "Data:"
vnstat --json a | jq '.interfaces[0].traffic.year'

# 6. Show total from JSON (all-time)
echo -e "\n=== Total (all-time) from JSON ==="
vnstat --json a | jq '.interfaces[0].traffic.total | (.rx + .tx) / 1099511627776' | awk '{printf "%.2f TiB\n", $1}'

# 7. Compare with CLI monthly output
echo -e "\n=== Monthly data from CLI ==="
vnstat -m

# 8. Compare with CLI yearly output
echo -e "\n=== Yearly data from CLI ==="
vnstat -y

# 9. Show vnstat version and config
echo -e "\n=== vnstat version ==="
vnstat --version

# 10. Check daily data retention config
echo -e "\n=== Check retention settings (if accessible) ==="
if [ -r /etc/vnstat.conf ]; then
    grep -E "DailyDays|MonthlyMonths|YearlyYears" /etc/vnstat.conf | grep -v "^#"
else
    echo "Cannot read /etc/vnstat.conf"
fi