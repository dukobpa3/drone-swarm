package backend

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dukobpa3/drone-swarm-sim/internal/config"
	"github.com/dukobpa3/drone-swarm-sim/internal/envcap"
)

const irisSDF = `<?xml version="1.0"?>
<sdf version="1.9"><model name="iris_with_ardupilot">
  <plugin name="ArduPilotPlugin" filename="ArduPilotPlugin">
    <fdm_port_in>9002</fdm_port_in>
    <control channel="0"><type>VELOCITY</type><p_gain>0.20</p_gain></control>
  </plugin>
</model></sdf>`

const fieldWorld = `<sdf version="1.9"><world name="field">
  <spherical_coordinates>
    <latitude_deg>37.5</latitude_deg>
    <longitude_deg>-122.25</longitude_deg>
    <elevation>38</elevation>
  </spherical_coordinates>
</world></sdf>`

// apLab writes an ArduPilot checkout, its build and an ardupilot_gazebo with
// the iris model, and a gz whose services succeed. It returns the lab YAML
// and the path of a world with spherical coordinates.
func apLab(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	for _, f := range []string{"copter.parm", "gazebo-iris.parm", "plane.parm"} {
		write(t, filepath.Join(root, "ardupilot", "Tools", "autotest", "default_params", f), "")
	}
	for _, bin := range []string{"arducopter", "arduplane"} {
		write(t, filepath.Join(root, "ardupilot", "build", "sitl", "bin", bin), "")
	}
	model(t, filepath.Join(root, "ardupilot_gazebo", "models"), "iris_with_ardupilot", irisSDF)
	write(t, filepath.Join(root, "gz", "bin", "gz"), "#!/bin/sh\necho 'data: true'\n")
	write(t, filepath.Join(root, "gz", "activate.sh"), "export PATH=\""+filepath.Join(root, "gz", "bin")+":$PATH\"\n")
	world := filepath.Join(root, "field.sdf")
	write(t, world, fieldWorld)
	return "run_dir: run\ngz: { activate: " + filepath.Join(root, "gz", "activate.sh") + " }\n" +
		"ardupilot: { root: " + filepath.Join(root, "ardupilot") + ", bin: " + filepath.Join(root, "ardupilot", "build", "sitl", "bin") +
		", gazebo: " + filepath.Join(root, "ardupilot_gazebo") + " }\n", world
}

func TestPlanArdupilotGz(t *testing.T) {
	labYAML, world := apLab(t)
	lab, prof := setup(t, labYAML, `
world: { sim: gz, name: `+world+` }
layout: { spacing: 2 }
groups:
  - { name: ap, autopilot: ardupilot, count: 2 }
`)
	specs, err := Plan(lab, prof)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(specs); got != "gz,ap-0,ap-1" {
		t.Fatalf("specs %s", got)
	}
	ap := specs[2]
	args := strings.Join(ap.Args, " ")
	for _, want := range []string{
		"--model JSON",
		"--instance 1",
		"--sysid 2",
		"--sim-address 127.0.0.1",
		// gz places the drone; SITL's home is the world's origin for all of them
		"--home 37.5000000,-122.2500000,38,0",
		"default_params/copter.parm,",
		"default_params/gazebo-iris.parm",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("args lack %q: %s", want, args)
		}
	}
	if ap.Before == nil || ap.Ready == nil {
		t.Fatal("an ardupilot drone in gz spawns its model and gates the next drone")
	}

	if err := ap.Before(context.Background()); err != nil {
		t.Fatal(err)
	}
	sdf, err := os.ReadFile(filepath.Join(lab.RunDir, prof.Name, "ap-1", "model.sdf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sdf), "<fdm_port_in>9012</fdm_port_in>") || strings.Contains(string(sdf), "9002") {
		t.Fatalf("ap-1's model must listen on its instance's port 9012:\n%s", sdf)
	}
}

func TestPlanArdupilotGzHomeFromProfile(t *testing.T) {
	labYAML, _ := apLab(t)
	world := filepath.Join(t.TempDir(), "flat.sdf")
	write(t, world, `<sdf version="1.9"><world name="flat"/></sdf>`)
	lab, prof := setup(t, labYAML, `
world: { sim: gz, name: `+world+`, home: { lat: 47.4, lon: 8.5, alt: 488 } }
groups:
  - { name: ap, autopilot: ardupilot, count: 1 }
`)
	specs, err := Plan(lab, prof)
	if err != nil {
		t.Fatal(err)
	}
	if args := strings.Join(specs[1].Args, " "); !strings.Contains(args, "--home 47.4000000,8.5000000,488,0") {
		t.Fatalf("world.home must win over the world: %s", args)
	}
}

func TestPlanArdupilotGzProblems(t *testing.T) {
	labYAML, world := apLab(t)
	flat := filepath.Join(t.TempDir(), "flat.sdf")
	write(t, flat, `<sdf version="1.9"><world name="flat"/></sdf>`)
	noGazebo, _, _ := strings.Cut(labYAML, ", gazebo:")
	noGazebo += " }\n"
	cases := []struct{ name, lab, profile, want string }{
		{"no gazebo in the lab", noGazebo, "", ""},
		{"unknown model", labYAML, `
world: { sim: gz, name: ` + world + ` }
groups: [ { name: ap, autopilot: ardupilot, model: x500, count: 1 } ]
`, `model "x500": swarmsim knows the frame parameters of iris_with_ardupilot`},
		{"vehicle does not match", labYAML, `
world: { sim: gz, name: ` + world + ` }
groups: [ { name: ap, autopilot: ardupilot, vehicle: plane, model: iris_with_ardupilot, count: 1 } ]
`, `is a copter, set vehicle: copter`},
		{"no origin", labYAML, `
world: { sim: gz, name: ` + flat + ` }
groups: [ { name: ap, autopilot: ardupilot, count: 1 } ]
`, `has no spherical_coordinates, set world.home`},
		{"payloads", labYAML, `
world: { sim: gz, name: ` + world + ` }
groups: [ { name: ap, autopilot: ardupilot, count: 1, payloads: [ { model: cam } ] } ]
`, `not supported for ardupilot yet`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			profile, want := c.profile, c.want
			if profile == "" {
				profile = "\nworld: { sim: gz, name: " + world + " }\ngroups: [ { name: ap, autopilot: ardupilot, count: 1 } ]\n"
				want = "ardupilot drones in gz need ardupilot.gazebo"
			}
			lab, prof := setup(t, c.lab, profile)
			if _, err := Plan(lab, prof); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("want %q, got %v", want, err)
			}
		})
	}
}

func TestArdupilotGzModelNeedsOnePort(t *testing.T) {
	labYAML, world := apLab(t)
	lab, prof := setup(t, labYAML, "\nworld: { sim: gz, name: "+world+" }\ngroups: [ { name: ap, autopilot: ardupilot, count: 1 } ]\n")
	sdf := filepath.Join(lab.ArduPilot.Gazebo, "models", "iris_with_ardupilot", "model.sdf")
	write(t, sdf, `<sdf><model name="iris"/></sdf>`)
	if _, err := Plan(lab, prof); err == nil || !strings.Contains(err.Error(), "want one ArduPilotPlugin with fdm_port_in, found 0") {
		t.Fatalf("want a port error, got %v", err)
	}
}

func TestWorldOrigin(t *testing.T) {
	sdf := filepath.Join(t.TempDir(), "w.sdf")
	write(t, sdf, fieldWorld)
	got, err := worldOrigin(sdf)
	if err != nil || got != (config.Home{Lat: 37.5, Lon: -122.25, Alt: 38}) {
		t.Fatalf("origin %+v, %v", got, err)
	}
	write(t, sdf, "<sdf")
	if _, err := worldOrigin(sdf); err == nil {
		t.Fatal("want a parse error")
	}
}

func TestSetStep(t *testing.T) {
	step := func(body string) error {
		env := gzBin(t, body, "")
		gz, err := envcap.LookPath(env, "gz")
		if err != nil {
			t.Fatal(err)
		}
		return setStep(context.Background(), gz, env, "w", 0.001)
	}
	if err := step("echo 'data: true'"); err != nil {
		t.Fatal(err)
	}
	if err := step("echo 'data: false'"); err == nil || !strings.Contains(err.Error(), `physics step of "w"`) {
		t.Fatalf("want a refusal, got %v", err)
	}
}

func TestPlanArdupilotGzStep(t *testing.T) {
	labYAML, world := apLab(t)
	lab, prof := setup(t, labYAML, `
world: { sim: gz, name: `+world+`, step_ms: 4 }
groups:
  - { name: ap, autopilot: ardupilot, count: 1, params: { FENCE_ENABLE: 1 } }
`)
	specs, err := Plan(lab, prof)
	if err != nil {
		t.Fatal(err)
	}
	if got := Step(prof, nil); got != 0.004 {
		t.Fatalf("step %g, want 0.004", got)
	}
	parm, err := os.ReadFile(filepath.Join(lab.RunDir, prof.Name, "ap-0", "swarmsim.parm"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"SCHED_LOOP_RATE 125\n", "FENCE_ENABLE 1\n"} {
		if !strings.Contains(string(parm), want) {
			t.Errorf("params lack %q:\n%s", want, parm)
		}
	}
	if err := specs[1].Before(context.Background()); err != nil {
		t.Fatal(err)
	}
	sdf, err := os.ReadFile(filepath.Join(lab.RunDir, prof.Name, "ap-0", "model.sdf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sdf), "<p_gain>0.05</p_gain>") {
		t.Fatalf("rotor gain not scaled to 4 ms:\n%s", sdf)
	}
}

func TestPlanStepProblems(t *testing.T) {
	labYAML, world := apLab(t)
	for _, c := range []struct{ profile, want string }{
		{"\nworld: { sim: gz, name: " + world + ", step_ms: 20 }\ngroups: [ { name: ap, autopilot: ardupilot, count: 1 } ]\n", "ardupilot drones fly at 10 ms steps at most"},
		{"\nworld: { sim: gz, name: " + world + ", step_ms: -1 }\ngroups: [ { name: ap, autopilot: ardupilot, count: 1 } ]\n", "give a positive step"},
		{"\nworld: { sim: none, step_ms: 4 }\ngroups: [ { name: e, autopilot: exec, sim: none, count: 1, command: [sleep, '1'] } ]\n", "no drone runs in gz"},
	} {
		lab, prof := setup(t, labYAML, c.profile)
		if _, err := Plan(lab, prof); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("want %q, got %v", c.want, err)
		}
	}
}

func TestStep(t *testing.T) {
	ap := []config.Drone{{Group: &config.Group{Autopilot: "ardupilot", Sim: "gz"}}}
	px := []config.Drone{{Group: &config.Group{Autopilot: "px4", Sim: "gz"}}}
	for _, c := range []struct {
		stepMS float64
		drones []config.Drone
		want   float64
	}{
		{0, px, 0},     // the world's own
		{0, ap, 0.001}, // ArduPilot as on a copter
		{4, ap, 0.004}, // the profile wins
		{2, px, 0.002},
	} {
		prof := &config.Profile{World: config.World{StepMS: c.stepMS}}
		if got := Step(prof, c.drones); got != c.want {
			t.Errorf("step_ms %g: %g, want %g", c.stepMS, got, c.want)
		}
	}
}

func TestWithLoopRate(t *testing.T) {
	if got := withLoopRate(nil, 0.001); len(got) != 0 {
		t.Fatalf("1 ms keeps the default loop: %v", got)
	}
	if got := withLoopRate(nil, 0.004)["SCHED_LOOP_RATE"]; got != 125 {
		t.Fatalf("4 ms: loop %g, want half of 250", got)
	}
	if got := withLoopRate(map[string]float64{"SCHED_LOOP_RATE": 100}, 0.004)["SCHED_LOOP_RATE"]; got != 100 {
		t.Fatalf("a profile's loop rate must win, got %g", got)
	}
}

func TestScaleGains(t *testing.T) {
	in := []byte("<p_gain>0.20</p_gain><i_gain>0</i_gain><p_gain> 1e-1 </p_gain>")
	if got := string(scaleGains(in, 0.25)); got != "<p_gain>0.05</p_gain><i_gain>0</i_gain><p_gain>0.025</p_gain>" {
		t.Fatalf("scaled %s", got)
	}
	if got := scaleGains(in, 1); string(got) != string(in) {
		t.Fatal("1 ms must leave the model as it is")
	}
}
