# Pomo

A terminal Pomodoro timer for tracking focused work. One process per work-and-rest period; nothing crosses session boundaries.

## Language

**Session**:
One run of `pomo` from launch to quit. The unit that the Markdown summary file describes; nothing persists across sessions.
_Avoid_: Run, invocation.

**Pomodoro**:
A single fixed-duration work block within a session. Has no display name or identity beyond its ordinal position; the user does not name pomodoros.
_Avoid_: Work block, focus block, interval, tomato.

**Break**:
A rest block that follows a pomodoro. Every Nth break is a **long break**; the others are **short breaks**.
_Avoid_: Rest, pause, intermission.

**Cycle**:
A run of N pomodoros (with their interleaved short breaks) ending in a long break. N is `pomodoros_per_cycle` (default 4).
_Avoid_: Round, set.

**Phase**:
The current state of the session — either inside a pomodoro or inside a break. Phases auto-transition in both directions; the user does not gate them with a keypress.
_Avoid_: Stage, mode.

**Task**:
A named unit of work the user is doing during the current pomodoro. A task has a name, a start time, and an end time. At most one task is active at a time; starting a new task ends the previous one.
_Avoid_: Activity, item, todo.

## Example dialogue

> **Dev**: When the pomodoro ends, do we wait for the user before starting the break?
>
> **Domain**: No. Phase transitions are automatic in both directions — pomodoro ends, break starts; break ends, the next pomodoro starts. The desktop notification is the user's signal. If they want to delay starting, they pause with `space`.
>
> **Dev**: And tasks — do they survive into the break?
>
> **Domain**: No. When a phase ends, the active task ends with it. If the next pomodoro continues the same work, the user starts a new task with the same name — it's a fresh task record.
>
> **Dev**: Got it. What about the long break — is it part of the cycle that just ended, or the next one?
>
> **Domain**: Part of the cycle that just ended. A cycle is `N pomodoros + N breaks`, where the Nth break is long. The next cycle starts with the pomodoro after that long break.
