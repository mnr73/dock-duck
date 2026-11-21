import {
    type MaybeRefOrGetter,
    type Ref,
    onBeforeUnmount,
    onMounted,
    shallowRef,
    toValue,
    watch,
} from "vue";

import {
    defaultKeymap,
    history,
    historyKeymap,
    indentWithTab,
} from "@codemirror/commands";
import { yaml } from "@codemirror/lang-yaml";
import {
    HighlightStyle,
    bracketMatching,
    indentOnInput,
    syntaxHighlighting,
} from "@codemirror/language";
import { type Diagnostic, lintGutter, setDiagnostics } from "@codemirror/lint";
import { highlightSelectionMatches, searchKeymap } from "@codemirror/search";
import { Compartment, EditorState, type Extension } from "@codemirror/state";
import {
    EditorView,
    drawSelection,
    highlightActiveLine,
    highlightActiveLineGutter,
    keymap,
    lineNumbers,
    placeholder as placeholderExt,
} from "@codemirror/view";
import { tags as t } from "@lezer/highlight";

import type { Problem } from "@/api/types";

/** Colours come from the app's CSS variables, so light/dark follow the theme without a reload. */
const theme = EditorView.theme({
    "&": {
        height: "100%",
        fontSize: "12.5px",
        color: "var(--color-fg)",
        backgroundColor: "var(--color-surface)",
    },
    ".cm-scroller": { fontFamily: "var(--font-mono)", lineHeight: "1.6" },
    ".cm-content": { caretColor: "var(--color-accent)", padding: "8px 0" },
    ".cm-gutters": {
        backgroundColor: "transparent",
        color: "var(--color-dim)",
        border: "none",
        paddingLeft: "8px",
    },
    ".cm-activeLine": {
        backgroundColor:
            "color-mix(in srgb, var(--color-accent) 7%, transparent)",
    },
    ".cm-activeLineGutter": {
        backgroundColor: "transparent",
        color: "var(--color-fg)",
    },
    "&.cm-focused": { outline: "none" },
    "&.cm-focused .cm-cursor": { borderLeftColor: "var(--color-accent)" },
    ".cm-selectionBackground, &.cm-focused .cm-selectionBackground, ::selection":
        {
            backgroundColor:
                "color-mix(in srgb, var(--color-accent) 28%, transparent) !important",
        },
    ".cm-lintRange-error": {
        backgroundImage: "none",
        textDecoration: "underline wavy var(--color-err)",
        textUnderlineOffset: "3px",
    },
    ".cm-lintRange-warning": {
        backgroundImage: "none",
        textDecoration: "underline wavy var(--color-warn)",
        textUnderlineOffset: "3px",
    },
    ".cm-tooltip": {
        border: "1px solid var(--color-line-strong)",
        backgroundColor: "var(--color-raised)",
        color: "var(--color-fg)",
        borderRadius: "6px",
    },
    ".cm-panels": {
        backgroundColor: "var(--color-raised)",
        color: "var(--color-fg)",
        borderTop: "1px solid var(--color-line)",
    },
    ".cm-searchMatch": {
        backgroundColor:
            "color-mix(in srgb, var(--color-warn) 30%, transparent)",
    },
});

const highlight = HighlightStyle.define([
    { tag: t.propertyName, color: "var(--color-info-fg)" },
    { tag: [t.string, t.content], color: "var(--color-ok-fg)" },
    { tag: [t.number, t.bool, t.null], color: "var(--color-warn-fg)" },
    { tag: t.comment, color: "var(--color-dim)", fontStyle: "italic" },
    { tag: [t.keyword, t.operator, t.meta], color: "var(--color-accent-text)" },
    { tag: [t.labelName, t.typeName], color: "var(--color-info-fg)" },
]);

export interface CodeEditorOptions {
    language: "yaml" | "plain";
    readOnly?: MaybeRefOrGetter<boolean>;
    placeholder?: string;
    onSave?: () => void;
}

/**
 * CodeMirror 6 bound to a string ref: v-model style, plus diagnostics.
 * Replacing `text` from outside (load, discard) resets the undo history.
 */
export function useCodeEditor(
    host: Ref<HTMLElement | null>,
    text: Ref<string>,
    opts: CodeEditorOptions,
) {
    const view = shallowRef<EditorView>();
    const readOnly = new Compartment();

    function extensions(): Extension[] {
        return [
            lineNumbers(),
            lintGutter(),
            highlightActiveLine(),
            highlightActiveLineGutter(),
            history(),
            drawSelection(),
            indentOnInput(),
            bracketMatching(),
            highlightSelectionMatches(),
            syntaxHighlighting(highlight),
            ...(opts.language === "yaml" ? [yaml()] : []),
            ...(opts.placeholder ? [placeholderExt(opts.placeholder)] : []),
            keymap.of([
                {
                    key: "Mod-s",
                    preventDefault: true,
                    run: () => {
                        opts.onSave?.();
                        return true;
                    },
                },
                indentWithTab,
                ...searchKeymap,
                ...historyKeymap,
                ...defaultKeymap,
            ]),
            readOnly.of(EditorState.readOnly.of(!!toValue(opts.readOnly))),
            theme,
            EditorView.updateListener.of((u) => {
                if (u.docChanged) {
                    const v = u.state.doc.toString();
                    if (v !== text.value) text.value = v;
                }
            }),
        ];
    }

    onMounted(() => {
        if (!host.value) return;
        view.value = new EditorView({
            parent: host.value,
            state: EditorState.create({
                doc: text.value,
                extensions: extensions(),
            }),
        });
    });
    onBeforeUnmount(() => view.value?.destroy());

    // outside -> editor (a plain typing round trip is ignored by the equality check)
    watch(text, (v) => {
        const ed = view.value;
        if (!ed || ed.state.doc.toString() === v) return;
        ed.setState(EditorState.create({ doc: v, extensions: extensions() }));
    });

    watch(
        () => !!toValue(opts.readOnly),
        (ro) =>
            view.value?.dispatch({
                effects: readOnly.reconfigure(EditorState.readOnly.of(ro)),
            }),
    );

    /** Show validation problems as squiggles + gutter markers; lines outside the document are clamped. */
    function setProblems(problems: Problem[]) {
        const ed = view.value;
        if (!ed) return;
        const doc = ed.state.doc;
        const diags: Diagnostic[] = problems
            .filter((p) => p.line)
            .map((p) => {
                const line = doc.line(
                    Math.min(Math.max(p.line!, 1), doc.lines),
                );
                return {
                    from: line.from,
                    to: Math.max(line.to, line.from),
                    severity: p.severity,
                    message: p.message,
                };
            });
        ed.dispatch(setDiagnostics(ed.state, diags));
    }

    function goToLine(n: number) {
        const ed = view.value;
        if (!ed) return;
        const line = ed.state.doc.line(
            Math.min(Math.max(n, 1), ed.state.doc.lines),
        );
        ed.dispatch({ selection: { anchor: line.from }, scrollIntoView: true });
        ed.focus();
    }

    function focus() {
        view.value?.focus();
    }

    return { view, setProblems, goToLine, focus };
}
