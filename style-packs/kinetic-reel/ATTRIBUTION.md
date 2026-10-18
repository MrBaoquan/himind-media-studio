# Attribution

This Style Pack is adapted from:

- https://github.com/tuzhechen2005/opus-video-skills
  - `skills/kinetic-reel/SKILL.md`
  - `skills/kinetic-reel/template/`
  - `skills/kinetic-reel/references/`
  - `skills/kinetic-reel/examples/`

The upstream repository is MIT licensed. `render.mjs` is derived from
John Heibel's ClaudeAnimationBase (MIT). See `LICENSE` for the full license text.

HiMind-specific changes:

- added `template/style-pack.json`
- fixed CLI integration with `media.workbench.project.import` / `preview.review`
- kept all rendering and authoring code in the Style Pack; the Workbench only
  manages project state, review artifacts, feedback and final artifacts.
