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
   negotiate who takes which task and keep their own spacing. GC only watches.

Stage 1 must not block stage 2: the task and capability model is the same in both; only the
place where the decision is made moves from GC to the companions.

## protocols

- An interface for what the services need from a drone: connect, telemetry (position,
  state, battery), capabilities, upload a mission, start, stop, return.
- The first implementation is a thin proxy over MAVLink: a mission upload is a MAVLink
  mission upload to the autopilot. Another transport or our own messages replace it later
  without changing GC.
- Messages between services and companions get a language-neutral schema (e.g. protobuf)
  as soon as a second language appears (C++ companion, Node GC).

## First steps

1. `protocols/`: the interface and the MAVLink implementation; checked against `sim/`
   (connect to every drone of a running swarm, read telemetry).
2. `gc/` as a command line tool: read a `.plan`, assign by requirements and offsets, upload
   to every drone, start. The four x500 of `sim/profiles/px4-sensors.yaml` (lidar, depth,
   thermal, plain) are the test bench: a thermal task must go to `thermal-0`.
3. GC's interface (map, swarm state).
4. `companion/`: reports capabilities, takes tasks; then stage 2.

## Open questions

- The task model: which actions and requirements are needed first; how a KML maps to tasks
  (a placemark is a point, a polygon an area?).
- What happens when no drone can do a task, or a drone drops out mid-mission.
- GC's language: Go (one language with `protocols`) or Node + Svelte (one language with
  the interface).
- The MAVLink library: gomavlib, MAVSDK (its Go bindings are thin), or our own on top of
  generated messages.
