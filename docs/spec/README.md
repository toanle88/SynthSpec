# Functional Specifications Overview

This directory contains functional specifications for SynthSpec's core features and processes.

## Specifications List

1. **[Interrogation Loop (The Oracle)](interrogation_loop.md)**: 
   Guides users through a single-question conversational loop until 100% confidence across four internal dimensions is achieved.
   
2. **[Asset Generation (Source-First Synthesis)](asset_generation.md)**: 
   Produces the final engineering deliverables from a locked domain source doc, then fans out downstream document generation in parallel within `synthspec-output/`.
