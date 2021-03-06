package logic

import (
	"bytes"
	"encoding/json"
	"os"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/jonaz/cron"
	"github.com/pborman/uuid"
	serverprotocol "github.com/stampzilla/stampzilla-go/nodes/stampzilla-server/protocol"
	"github.com/stampzilla/stampzilla-go/protocol"
)

// Schedular that schedule ruleActions
type Scheduler struct {
	tasks         []Task
	Nodes         *serverprotocol.Nodes `inject:""`
	ActionService *ActionService        `inject:""`
	Logic         *Logic                `inject:""`
	Cron          *cron.Cron
	sync.RWMutex
}

func NewScheduler() *Scheduler {
	scheduler := &Scheduler{}
	scheduler.Cron = cron.New()
	return scheduler
}

func (s *Scheduler) Start() {
	log.Info("Starting Scheduler")

	s.loadFromFile("schedule.json")
	s.Cron.Start()
}

func (s *Scheduler) Reload() {
	//TODO verify the the new JSON is valid before unloading the existing schedule
	s.Lock()
	s.tasks = nil
	s.Unlock()
	for _, job := range s.Cron.Entries() {
		s.Cron.RemoveJob(job.Id)
	}
	s.Cron.Stop()
	s.Start()
}

func (s *Scheduler) Tasks() []Task {
	s.RLock()
	defer s.RUnlock()
	return s.tasks
}

func (s *Scheduler) AddTask(name string) Task {
	var err error

	task := &task{Name_: name, Uuid_: uuid.New()}
	if s.Logic != nil {
		task.ActionProgressChan = s.Logic.ActionProgressChan
	}
	task.actionService = s.ActionService

	//task.nodes = s.Nodes
	task.cron = s.Cron
	if err != nil {
		log.Error(err)
	}
	s.Lock()
	s.tasks = append(s.tasks, task)
	s.Unlock()
	return task
}

func (s *Scheduler) RemoveTask(uuid string) error {
	s.Lock()
	defer s.Unlock()
	for i, task := range s.tasks {
		if task.Uuid() == uuid {
			s.Cron.RemoveJob(task.CronId())
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			return nil
		}
	}
	return nil
}
func (s *Scheduler) CreateExampleFile() {
	task := s.AddTask("Test1")
	cmd := &protocol.Command{Cmd: "testCMD"}
	action := &action{
		Commands: []Command{NewCommand(cmd, "simple")},
	}
	task.AddAction(action)
	task.Schedule("0 * * * * *")

	s.saveToFile("schedule.json")
}

func (s *Scheduler) saveToFile(filepath string) {
	configFile, err := os.Create(filepath)
	if err != nil {
		log.Error("creating config file", err.Error())
	}
	var out bytes.Buffer
	b, err := json.Marshal(s.tasks)
	if err != nil {
		log.Error("error marshal json", err)
	}
	json.Indent(&out, b, "", "\t")
	out.WriteTo(configFile)
}

func (s *Scheduler) loadFromFile(filepath string) {
	log.Info("Loading schedule from json file")

	configFile, err := os.Open(filepath)
	if err != nil {
		log.Warn("opening config file", err.Error())
		return
	}

	//TODO we can use normal task here if we refactor some stuff :)
	type localTasks struct {
		Name     string   `json:"name"`
		Uuid     string   `json:"uuid"`
		Actions  []string `json:"actions"`
		CronWhen string   `json:"when"`
	}

	var tasks []*localTasks
	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&tasks); err != nil {
		log.Error(err)
	}

	for _, task := range tasks {
		t := s.AddTask(task.Name)

		//Set the uuid from json if it exists. Otherwise use the generated one
		if task.Uuid != "" {
			t.SetUuid(task.Uuid)
		}
		for _, uuid := range task.Actions {
			a := s.ActionService.GetByUuid(uuid)
			if a == nil {
				log.Errorf("Could not find action %s. Skipping adding it to task.\n", uuid)
				continue
			}
			t.AddAction(a)
		}
		//Schedule the task!
		t.Schedule(task.CronWhen)
	}

}
