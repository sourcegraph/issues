#!/bin/bash

set -ex

declare mermaid_diagrams=(
  definitions
  references
  resolve-page
  hover
  upload
  extension-definitions
  extension-references
  extension-hover
)

# Install mermaid util
yarn
mermaid="../../../../node_modules/.bin/mmdc"

# Generate mermaid diagrams
for diagram in "${mermaid_diagrams[@]}"; do
  "${mermaid}" -i "${diagram}.mermaid" -o "${diagram}.svg"

  # Make the generated id deterministic so CI won't see superflouous changes
  sed -i '' "s/mermaid-[0-9]\{1,\}/mermaid-${diagram}/g" "${diagram}.svg"
done
