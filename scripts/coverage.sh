#!/bin/bash

# coverage.sh - Generate code coverage reports for GoLikeit
# This script generates HTML coverage reports and validates thresholds

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_DIR="coverage"
REPORT_FILE="$COVERAGE_DIR/coverage.out"
HTML_REPORT="$COVERAGE_DIR/index.html"
THRESHOLD=80

# Create coverage directory
mkdir -p "$COVERAGE_DIR"

echo "Generating coverage report..."

# Run tests with coverage
go test -race -coverprofile="$REPORT_FILE" -covermode=atomic ./... 2>&1 || true

# Check if coverage file was generated
if [ ! -f "$REPORT_FILE" ]; then
    echo -e "${RED}Error: Coverage report not generated${NC}"
    exit 1
fi

# Generate HTML report
echo "Generating HTML report..."
go tool cover -html="$REPORT_FILE" -o "$HTML_REPORT"

# Generate function-level report
echo "Generating function coverage report..."
go tool cover -func="$REPORT_FILE" > "$COVERAGE_DIR/coverage.txt"

# Calculate overall coverage
echo ""
echo "=== Coverage Summary ==="
COVERAGE=$(go tool cover -func="$REPORT_FILE" | grep total | awk '{print $3}')
echo "Overall coverage: $COVERAGE"

# Extract percentage
COVERAGE_NUM=$(echo "$COVERAGE" | sed 's/%//')

# Check threshold
echo ""
echo "=== Threshold Check ==="
echo "Minimum required: ${THRESHOLD}%"
echo "Current coverage: ${COVERAGE_NUM}%"

if (( $(echo "$COVERAGE_NUM >= $THRESHOLD" | bc -l) )); then
    echo -e "${GREEN}✓ Coverage threshold met${NC}"
    THRESHOLD_MET=0
else
    echo -e "${RED}✗ Coverage below threshold${NC}"
    THRESHOLD_MET=1
fi

# Package-level coverage
echo ""
echo "=== Package Coverage ==="
go tool cover -func="$REPORT_FILE" | grep -E "^github.com" | while read -r line; do
    PKG=$(echo "$line" | awk '{print $1}')
    COV=$(echo "$line" | awk '{print $3}')
    echo "$PKG: $COV"
done

# Generate package summary
echo ""
echo "=== Package Summary (JSON) ==="
cat > "$COVERAGE_DIR/coverage.json" <<EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "total_coverage": "$COVERAGE_NUM",
  "threshold": $THRESHOLD,
  "threshold_met": $([ $THRESHOLD_MET -eq 0 ] && echo "true" || echo "false"),
  "report_file": "$REPORT_FILE",
  "html_report": "$HTML_REPORT"
}
EOF

cat "$COVERAGE_DIR/coverage.json"

# Generate badge (if COVERAGE_BADGE env var is set or requested)
if [ "${GENERATE_BADGE:-}" = "true" ] || [ -n "$COVERAGE_BADGE" ]; then
    echo ""
    echo "=== Generating Badge ==="

    # Determine badge color
    if (( $(echo "$COVERAGE_NUM >= 90" | bc -l) )); then
        COLOR="brightgreen"
    elif (( $(echo "$COVERAGE_NUM >= 80" | bc -l) )); then
        COLOR="green"
    elif (( $(echo "$COVERAGE_NUM >= 70" | bc -l) )); then
        COLOR="yellow"
    elif (( $(echo "$COVERAGE_NUM >= 60" | bc -l) )); then
        COLOR="orange"
    else
        COLOR="red"
    fi

    # Generate SVG badge using shields.io style
    cat > "$COVERAGE_DIR/badge.svg" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="108" height="20" role="img" aria-label="coverage: ${COVERAGE_NUM}%" viewBox="0 0 108 20">
  <title>coverage: ${COVERAGE_NUM}%</title>
  <linearGradient id="s" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r">
    <rect width="108" height="20" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#r)">
    <rect width="61" height="20" fill="#555"/>
    <rect x="61" width="47" height="20" fill="${COLOR}"/>
    <rect width="108" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="110">
    <text x="315" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="510">coverage</text>
    <text x="315" y="140" transform="scale(.1)" textLength="510">coverage</text>
    <text x="835" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="370">${COVERAGE_NUM}%</text>
    <text x="835" y="140" transform="scale(.1)" textLength="370">${COVERAGE_NUM}%</text>
  </g>
</svg>
EOF

    echo "Badge generated: $COVERAGE_DIR/badge.svg"
fi

echo ""
echo "=== Reports Generated ==="
echo "Coverage file: $REPORT_FILE"
echo "HTML report: $HTML_REPORT"
echo "Text report: $COVERAGE_DIR/coverage.txt"
echo "JSON report: $COVERAGE_DIR/coverage.json"
[ -f "$COVERAGE_DIR/badge.svg" ] && echo "Badge: $COVERAGE_DIR/badge.svg"

exit $THRESHOLD_MET
