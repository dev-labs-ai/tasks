# PRD --- Todo List MVP

**Status:** Draft
**Last updated:** 2026-09-25

Technical constraints and stack: see
[web-system-stack-specification.md](web-system-stack-specification.md).
This document defines what the product does and why, not how it is
built.

------------------------------------------------------------------------

## 1. Problem

People need a simple place to write down what they have to do and keep
track of what is done, reachable from any browser and private to them.

## 2. Target user

An individual managing their own tasks. One person per account; no
teams, no sharing.

## 3. Goals

-   A new user can register and manage tasks without help.
-   A user's tasks are private and persist across sessions and devices.
-   The MVP covers only what is needed to manage a task list; anything
    else waits for real usage.

### Definition of done

-   Every acceptance criterion in section 5 is covered by an automated
    test.
-   The main flow in section 6 passes as an E2E test.
-   Backend tests prove that no user can read or change another user's
    tasks.

------------------------------------------------------------------------

## 4. Non-goals

Out of scope for the MVP:

-   Due dates and reminders
-   Priorities
-   Tags or multiple lists
-   Subtasks
-   Drag-and-drop reordering
-   Search
-   Sharing
-   Email verification and password reset
-   Offline mode

------------------------------------------------------------------------

## 5. Requirements

### 5.1 Authentication

**FR-1 --- Register.** As a visitor, I want to create an account so that
my tasks are private.

-   Email must be valid and not already registered.
-   Password must have 8 to 64 characters.
-   After registering, the user is logged in.

**FR-2 --- Log in.** As a user, I want to log in so that I can reach my
tasks.

-   Valid credentials take the user to their task list.
-   A wrong email or a wrong password shows the same generic message,
    without saying which one was wrong.
-   The session survives a page reload.
-   A session lasts 1 day; after that the user logs in again.
-   Each email allows 5 login attempts per 15 minutes. Further attempts
    are refused with a message saying when to try again. A successful
    login resets the count.

**FR-3 --- Log out.** As a user, I want to log out so that nobody else
using the browser can see my tasks.

-   Logging out ends the session.
-   Opening the task list while logged out redirects to the login page.

**FR-4 --- Ownership.** As a user, I want my tasks to be visible only to
me.

-   A user only ever sees their own tasks.
-   Reading, editing, completing, or deleting another user's task fails
    as if the task did not exist.
-   Enforced by the backend, not only by the UI.

### 5.2 Tasks

**FR-5 --- Create.** As a user, I want to add a task so that I do not
forget it.

-   The title is required; a title of only whitespace is rejected.
-   Leading and trailing whitespace is trimmed.
-   The title has at most 200 characters.
-   A new task starts as pending and appears at the top of the list.

**FR-6 --- List.** As a user, I want to see my tasks so that I know what
is left to do.

-   Tasks are ordered newest first.
-   Each task shows its title and whether it is completed.

**FR-7 --- Edit.** As a user, I want to change a task's title so that I
can fix or refine it.

-   The same title rules as FR-5 apply.

**FR-8 --- Complete or reopen.** As a user, I want to mark a task as done
so that I can see my progress.

-   A pending task can be completed, and a completed task can be
    reopened.
-   Completed tasks are visually distinct from pending ones.

**FR-9 --- Delete.** As a user, I want to remove a task I no longer
need.

-   Deletion is permanent.

**FR-10 --- Filter.** As a user, I want to see only pending or only
completed tasks so that I can focus.

-   Filters: all, pending, completed. The default is all.
-   The selected filter is kept in the URL and survives a reload.

### 5.3 UI states

**FR-11 --- Empty, loading, and error states.**

-   An empty list shows a message prompting the user to create a task.
-   Loading is indicated while data is fetched.
-   An error shows a readable message and keeps what the user typed, so
    they can retry.

------------------------------------------------------------------------

## 6. Main user flow

Register → create a task → complete it → filter by completed → delete
it → log out.

------------------------------------------------------------------------

## 7. Non-functional requirements

-   **Security** --- authentication and authorization live in the
    backend (specification section 9). Passwords are stored hashed,
    never in plain text.
-   **Validation** --- in the frontend for fast feedback and in the
    backend as the definitive layer (specification section 10).
-   **Errors** --- never expose internal details to the user
    (specification section 8).
-   **Responsive** --- usable on current desktop and mobile browsers.
-   **Accessibility** --- every action can be done with the keyboard.

------------------------------------------------------------------------

## 8. Open questions

-   Is 200 characters the right maximum for a title?
-   Should deleting a task ask for confirmation or offer undo?
-   Without password reset, a user who forgets their password loses the
    account. Is that acceptable for the MVP?
