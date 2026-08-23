# ASCII Diagrams

Read this when drawing a flow or structure diagram.

- chars: only `|` `v` `+--` `->`
- avoid: arrows (`→` `⇒` `➜`), box-drawing (`─` `│` `┌` `└` `├` `┬`), bullet glyphs (`•` `▪`)
- fence: always ` ```text ` (never bare ` ``` `) — stops renderers from syntax-highlighting or reflowing the layout
- caption: one sentence stating what the diagram depicts, placed directly above or below it
- labels: inline on the arrow if <=4 words; longer explanations go to a numbered/bulleted legend below
