# Roadmap

The order of the work. [design.md](design.md) says what the swarm is meant to become; this
file says what comes first and when a phase is done.

The rule for the order: **isolate a low-level problem before adding the coordination that
would hide it.** Offsets from GC would keep drones apart in the bench and say nothing about
what happens when two of them meet anyway.

## Phase 0: the bench (done)

`sim/` starts a swarm from a profile: PX4 SITL in gz, each drone with its own sensors
(stock PX4 variants, or a base model with payloads built by swarmsim), all visible in QGC
as separate vehicles. Missions are uploaded per drone from QGC.

The same mission uploaded to every drone makes them collide, again and again; the drone
with a lidar collides too. That is the problem phase 1 starts from.

## Phase 1: collision avoidance in the autopilot

What the autopilots do on their own about obstacles and other drones, before anything of
ours. **Not** in this phase: offsets, formations, slot assignment, task allocation, our own
protocol or our own avoidance planner (unless the phase shows it is needed).

### Starting points (from the code, to be confirmed in the sim)

PX4 as built in the lab (`v1.18.0-beta1-1258`):

- `gz_x500_lidar_2d`: gz's `LaserScan` goes through `GZBridge` into `obstacle_distance`
  (body FRD), which is what collision prevention reads.
- Collision prevention is off by default (`CP_DIST = -1`); neither the airframe nor our
  profiles set it.
- It acts only in **Position mode** (it limits the stick setpoint). Mission (AUTO) does
  not use it, and this PX4 has no interface for an external obstacle-avoidance planner in
  AUTO any more. So in a mission nothing in PX4 stops a collision: the collisions we see are
  the expected behaviour, not a broken sensor.

ArduPilot (in the lab, not yet run in the swarm):

- `AC_Avoidance` with `AP_Proximity`: fences and proximity sensors, and path planning
  around obstacles (`OA_TYPE`: BendyRuler, Dijkstra) in AUTO and GUIDED.
- `AP_Avoidance`: avoids other vehicles from their reported positions (ADS-B or MAVLink),
  without sensors; the one built-in mechanism aimed at drone-to-drone conflicts.

### Experiments

The regression scenario: four drones, one mission with the same waypoints for all, no
offsets.

- **A. Sensor pipeline.** One drone with a lidar: does gz publish the scan, does PX4 get
  it, does `obstacle_distance` appear with sane frame, range and field of view? With
  `CP_DIST` set, does it hold the drone off a wall in Position mode? A lidar in the model
  proves nothing on its own.
- **B. Static obstacle.** One drone and a fixed obstacle: Position mode and Mission. The
  outcome to record is whatever the autopilot does (stops, slides, goes around, nothing).
- **C. Four drones.** Four with lidars, the same mission: do they see each other (a 2D
  scan sees only its own height), react to a moving drone, deadlock or oscillate, carry
  on after a conflict?
- **D. ArduPilot**, only if PX4 has no answer for AUTO: B and C with `OA_TYPE` and with
  `AP_Avoidance`, to compare what it does and what it costs to set up. No migration
  without a reason from these results.

The bench needs for this: a profile with four lidar drones, a world with a static
obstacle, and a way to set autopilot parameters per profile.

### Measured

Per run: collisions, the minimum distance between drones, avoidance interventions
(stops), the flight modes and failsafes, missions finished, deadlocks. From the logs the
autopilots already write (ULog for PX4), with the same start conditions every run, so a
change in configuration can be compared with the run before it.

### Done when

- It is established, from runs, whether the built-in avoidance works with our sensors and
  flight modes, for one drone and for four.
- Its limits are written down.
- There is a decision, with reasons, on where avoidance lives: the flight controller, the
  companion, or both. For this phase it is enough that a collision is prevented
  predictably, even if the drone stops and the mission does not finish.

## Phase 2: one mission for the swarm, GC assigns

Stage 1 of the allocation in [design.md](design.md): `protocols/` over MAVLink, then `gc/`
as a command line tool that reads a `.plan` or KML, matches tasks to drones by equipment and
gives them offsets. Its test: a thermal task goes to `thermal-0` of
`sim/profiles/px4-sensors.yaml`. Then GC's interface.

## Phase 3: companion

`companion/` reports capabilities and takes tasks; the navigation part (local planning,
dynamic obstacles) comes in here if phase 1 decided it belongs to the companion.

## Later

Stage 2 of the allocation and the cluster runtime from [design.md](design.md): the
companions plan among themselves, GC is one participant, state is replicated with traffic
priorities.
