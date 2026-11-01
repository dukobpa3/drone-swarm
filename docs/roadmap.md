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
- ArduPilot SITL and `ardupilot_gazebo` are built against `drone-lab/gz`.
- **ArduPilot in gz from swarmsim** (`sim/profiles/ap-4x-gz.yaml`): swarmsim writes each
  drone a copy of the model with the `ArduPilotPlugin` on its own port (9002 + 10 ×
  instance) and spawns it, starts SITL with the frame's parameters and the world's origin
  as home, and sets the physics step to 1 ms. Four iris have flown in baylands at 0.86 of
  real time, once the iris's collision mesh became a box (as a mesh it held gz at 0.09).

Left:

- **Mixed profiles**: PX4 and ArduPilot groups in one world. With ArduPilot drones the world
  runs at 1 ms, and PX4 takes that: four x500 in baylands at 1 ms booted, armed and took off
  as at 4 ms, gz at 0.93 of real time (1.0 at 4 ms). The cost is CPU: gz 33% → 74%, the
  four PX4 79% → 103%. Sysids and ports unique across both.

  ArduPilot can run at PX4's 4 ms too, with two changes, checked in baylands:
  - the iris's rotor speed controller (a P gain on the joint) goes unstable at 4 ms:
    `p_gain · dt / I` is 1.19 at 1 ms and 4.8 at 4 ms (stable below 2), so the rotors
    never spin up. `p_gain` 0.2 → 0.05 gives each step the same dynamics as 0.2 at 1 ms;
  - copter's loop runs once per physics frame, 250 Hz, and arming wants it at 90% of
    `SCHED_LOOP_RATE` and the gyro at 1.8× of it. `SCHED_LOOP_RATE 125` passes both;
    250 needs the INS checks skipped (`ARMING_SKIPCHK 16`); 200 runs at 166 Hz and fails.

  | iris, one drone | hover | 30 m north | four drones: gz / four SITL CPU |
  |---|---|---|---|
  | 1 ms, loop 400 | 10.00 ± 0.005 m, tilt 0.4° | 6.6 s, overshoot 0.09 m | 145% / 78% |
  | 4 ms, loop 125 | 10.00 ± 0.014 m, tilt 2.7° | 6.7 s, overshoot 0.10 m | 40% / 53% |
  | 4 ms, loop 250 | 10.00 ± 0.013 m, tilt 3.2° | 6.5 s, overshoot 0.10 m | 38% / 53% |

  So 4 ms costs a third of the CPU and flies slightly looser, with a control loop slower
  than a real copter's 400 Hz. 1 ms stays the default; 4 ms is the option when CPU limits
  the number of drones.
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
