# Swarm design

What the swarm is meant to do and how the services split the work. This is the plan,
not a description of code that exists: each section says what is built and what is not.

## The idea

An operator plans **one mission for the whole swarm**: points, areas, what to photograph,
where to fly. Ground control loads it "into the swarm", and the swarm decides who does what:
a drone takes the tasks its equipment allows (a thermal task needs a thermal camera), and the
drones are placed so they do not fly into each other.

## Services

| | Role | Language | Built |
|---|---|---|---|
| `sim/` | Starts a simulated swarm (gz world + autopilots) from a profile. | Go | yes |
| `protocols/` | How the services and the drones talk. An abstraction with replaceable implementations; the first one is a proxy over MAVLink. | Go first | no |
| `gc/` | Ground control: loads the mission, splits it into tasks, hands them to the drones, shows the swarm. | Go or Node + Svelte | no |
| `companion/` | Runs on each drone next to the autopilot: knows the drone's equipment, takes tasks, later coordinates with the others. | C++ ideally, Go for a start | no |

## Mission

- **Input.** A QGroundControl `.plan` or a KML file. There is no mission editor of our own
  for now: the operator plans in QGC or Google Earth.
- **Model.** The file becomes the swarm's mission: a list of **tasks**. A task is a place
  (point, path, area) and an action (fly through, photograph, survey), with **requirements**:
  the equipment it needs (camera, thermal, lidar, depth).
- **Capabilities.** Each drone reports what it carries. In the simulation that is known from
  the profile (`payloads`, the PX4 model variant); on a real drone the companion reports it.

## Who does what (allocation)

Two stages, the first one simple on purpose:

1. **GC assigns.** GC matches tasks to drones by requirements, then by distance. Where
   drones share a path or an area, GC gives each one an **offset** (or its own strip of the
   area), so they keep apart without talking to each other. Each drone gets an ordinary
   autopilot mission.
2. **The swarm assigns itself.** The mission goes to the swarm as a whole; the companions
   negotiate who takes which task, plan their own paths and keep their own spacing, and go
   on when GC is out of reach. GC is one participant of the cluster: it can do the heavy
   computing, configure missions and watch them, but the swarm does not depend on it.
   State and results are replicated between participants, with priorities so the
   critical links are not flooded.

Stage 1 must not block stage 2: the task and capability model is the same in both; only the
place where the decision is made moves from GC to the companions.

## Layers on a drone

| Layer | Does |
|---|---|
| Flight controller (PX4 or ArduPilot) | Stabilisation, low-level control, failsafes, the collision prevention it supports |
| Companion: navigation | Perception, local planning, dynamic obstacles, where the flight controller is not enough |
| Companion: cluster runtime | The shared mission, task allocation, capabilities, replication, coordination |

Logical layers, not processes. What goes into the flight controller and what into the
companion is decided by phase 1 of the [roadmap](roadmap.md): no avoidance of our own
before the built-in one is measured.

## protocols

- An interface for what the services need from a drone: connect, telemetry (position,
  state, battery), capabilities, upload a mission, start, stop, return.
- The first implementation is a thin proxy over MAVLink: a mission upload is a MAVLink
  mission upload to the autopilot. Another transport or our own messages replace it later
  without changing GC.
- Messages between services and companions get a language-neutral schema (e.g. protobuf)
  as soon as a second language appears (C++ companion, Node GC).

## Order of the work

[roadmap.md](roadmap.md). In short: first what the autopilots do about collisions on their
own (phase 1), then `protocols/` and `gc/` with stage 1 (phase 2), then `companion/`.

## Open questions

- The task model: which actions and requirements are needed first; how a KML maps to tasks
  (a placemark is a point, a polygon an area?).
- What happens when no drone can do a task, or a drone drops out mid-mission.
- GC's language: Go (one language with `protocols`) or Node + Svelte (one language with
  the interface).
- The MAVLink library: gomavlib, MAVSDK (its Go bindings are thin), or our own on top of
  generated messages.
