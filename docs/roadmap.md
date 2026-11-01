# Roadmap

The order of the work. [design.md](design.md) says what the swarm is meant to become; this
file says what comes first and when a phase is done.

The rule for the order: **isolate a low-level problem before adding the coordination that
would hide it.** Offsets from GC would keep drones apart in the bench and say nothing about
what happens when two of them meet anyway.

## Phase 0: the bench

A swarm of any mix of PX4 and ArduPilot drones in one gz world, from one profile: two PX4
and two ArduPilot, say, each with its own sensors, all in QGC as separate vehicles.

Done:

- `sim/` starts PX4 SITL in gz from a profile, each drone with its own sensors (stock PX4
  variants, or a base model with payloads built by swarmsim). Missions are uploaded per
  drone from QGC. The same mission on every drone makes them collide, again and again; the
  drone with a lidar collides too. That is the problem phase 1 starts from.
- ArduPilot SITL and `ardupilot_gazebo` are built against `drone-lab/gz`; three ArduPilot
  iris have flown in gz, started by hand.

Left:

- **ArduPilot in gz from swarmsim.** Its backend runs only the builtin physics today. In gz
  the drones do not spawn themselves as PX4's do: swarmsim has to write each drone's model
  with the `ArduPilotPlugin` on its own port (9002 + 10 × instance), put it in the world,
  start SITL with the frame's parameters and the world's home.
- **Mixed profiles**: PX4 and ArduPilot groups in one world. One set of gz system plugins
  and one physics step for both (PX4's worlds run 4 ms, ArduPilot's 1 ms; to be checked
  what each tolerates), sysids and ports unique across both.
- **Sensors on ArduPilot drones**, a lidar first: see phase 1.

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

ArduPilot (built in the lab, flown in gz by hand):

- `AC_Avoidance` with `AP_Proximity`: fences and proximity sensors, and path planning
  around obstacles (`OA_TYPE`: BendyRuler, Dijkstra) in AUTO and GUIDED.
- `AP_Avoidance`: avoids other vehicles from their reported positions (ADS-B or MAVLink),
  without sensors; the one built-in mechanism aimed at drone-to-drone conflicts.
- A lidar reaches it from gz only as up to six rangefinders: `ardupilot_gazebo` sends
  `rng_1`..`rng_6` in its JSON, one distance each, not a scan. `AP_Proximity` can build its
  sectors from rangefinders. A full scan needs `OBSTACLE_DISTANCE` over MAVLink from a
  process that reads gz, which is a companion's job.

### Steps

The regression scenario: four drones, one mission with the same waypoints for all, no
offsets, on the phase 0 bench. ArduPilot goes first: its avoidance works in AUTO, PX4's
does not. PX4 stays as the baseline to compare with.

1. **One lidar drone among the others.** One drone with a lidar, three without, the
   same mission. The lidar drone must not run into the others; the others still fly blind,
   so a collision they cause is expected and recorded apart. On the way:
   - the sensor pipeline: gz publishes the scan, the autopilot gets it as proximity data
     with sane frame, range and field of view (a lidar in the model proves nothing on its
     own; a 2D scan sees only its own height);
   - a static obstacle first, then moving drones;
   - Position mode (PX4 `CP_DIST`) and AUTO (ArduPilot `OA_TYPE`, `AVOID_*`);
   - `AP_Avoidance`, which needs other drones' positions rather than a sensor.
2. **Lidars on all four.** Only once step 1 works: do they deadlock or oscillate when
   each gives way to the others, do they carry on after a conflict, what happens when
   several meet at once.

The bench needs for this: profiles with one and with four lidar drones, and a world with a
static obstacle.

### Measured

Per run: collisions, the minimum distance between drones, avoidance interventions
(stops), the flight modes and failsafes, missions finished, deadlocks. From the logs the
autopilots already write (ULog for PX4, dataflash for ArduPilot), with the same start conditions every run, so a
change in configuration can be compared with the run before it.

### Done when

- It is established, from runs, whether the built-in avoidance works with our sensors and
  flight modes: for one lidar drone among blind ones, then for four lidar drones.
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
