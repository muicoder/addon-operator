package module_manager

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/magiconair/properties/assert"

	"github.com/flant/antiopa/helm"
	"github.com/flant/antiopa/kube_config_manager"
	"github.com/flant/antiopa/utils"
)

func runInitModulesIndex(t *testing.T, mm *MainModuleManager, subPath string) {
	initTempAndWorkingDirectories(t, filepath.Join("init_modules_index", subPath))

	if err := mm.initModulesIndex(); err != nil {
		t.Fatal(err)
	}
}

func runInitGlobalHooks(t *testing.T, mm *MainModuleManager, subPath string) {
	initTempAndWorkingDirectories(t, filepath.Join("init_global_hooks", subPath))

	if err := mm.initGlobalHooks(); err != nil {
		t.Fatal(err)
	}
}

func initTempAndWorkingDirectories(t *testing.T, subPath string) {
	_, testFile, _, _ := runtime.Caller(0)
	testDirectory := filepath.Dir(testFile)
	WorkingDir = filepath.Join(testDirectory, "testdata", subPath)

	var err error
	TempDir, err = ioutil.TempDir("", "antiopa-")
	if err != nil {
		t.Fatal(err)
	}
}

func TestMainModuleManager_globalConfigValues(t *testing.T) {
	mm := &MainModuleManager{}
	runInitModulesIndex(t, mm, "test_global_config_values")

	expectedValues := utils.Values{
		"a": 1.0,
		"b": 2.0,
		"c": 3.0,
		"d": []interface{}{"a", "b", "c"},
	}

	if !reflect.DeepEqual(mm.globalConfigValues, expectedValues) {
		t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectedValues, mm.globalConfigValues)
	}
}

func TestMainModuleManager_globalModulesConfigValues(t *testing.T) {
	mm := &MainModuleManager{}
	runInitModulesIndex(t, mm, "test_global_modules_config_values")

	var expectations = []struct {
		moduleName string
		values     utils.Values
	}{
		{
			moduleName: "with-values-1",
			values:     utils.Values{"a": 1.0, "b": 2.0, "c": 3.0},
		},
		{
			moduleName: "with-values-2",
			values:     utils.Values{"a": []interface{}{1.0, 2.0, map[string]interface{}{"b": 3.0}}},
		},
	}

	for _, expectation := range expectations {
		t.Run(expectation.moduleName, func(t *testing.T) {
			if !reflect.DeepEqual(mm.globalModulesConfigValues[expectation.moduleName], expectation.values) {
				t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectation.values, mm.globalModulesConfigValues[expectation.moduleName])
			}
		})
	}
}

func TestMainModuleManager_GetModule2(t *testing.T) {
	mm := &MainModuleManager{}
	runInitModulesIndex(t, mm, "test_get_module")

	var expectations = []*Module{
		{
			Name:          "module",
			Path:          filepath.Join(WorkingDir, "modules/000-module"),
			DirectoryName: "000-module",
			moduleManager: mm,
		},
	}

	for _, expectation := range expectations {
		t.Run(expectation.Name, func(t *testing.T) {
			module, err := mm.GetModule(expectation.Name)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(module, expectation) {
				t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectation, module)
			}
		})
	}
}

func TestMainModuleManager_GetModuleNamesInOrder2(t *testing.T) {
	mm := &MainModuleManager{}
	runInitModulesIndex(t, mm, "test_get_module_names_in_order")

	expectedModules := []string{
		"module-c",
		"module-a",
		"module-b",
	}

	modulesInOrder := mm.GetModuleNamesInOrder()
	if !reflect.DeepEqual(expectedModules, modulesInOrder) {
		t.Errorf("\n[EXPECTED]: %s\n[GOT]: %s", expectedModules, modulesInOrder)
	}
}

func TestMainModuleManager_GetModuleHook2(t *testing.T) {
	mm := &MainModuleManager{}
	runInitModulesIndex(t, mm, "test_get_module_hook")

	createModuleHook := func(moduleName, name string, bindings []BindingType, orderByBindings map[BindingType]float64, schedules []ScheduleConfig) *ModuleHook {
		moduleHook := mm.newModuleHook()
		moduleHook.Name = name
		moduleHook.moduleManager = mm

		var err error
		if moduleHook.Module, err = mm.GetModule(moduleName); err != nil {
			t.Fatal(err)
		}

		moduleHook.Path = filepath.Join(WorkingDir, "modules", name)
		moduleHook.Schedules = schedules
		moduleHook.Bindings = bindings
		moduleHook.OrderByBinding = orderByBindings

		return moduleHook
	}

	expectations := []struct {
		moduleName     string
		name           string
		bindings       []BindingType
		orderByBinding map[BindingType]float64
		schedule       []ScheduleConfig
	}{
		{
			"all-bindings",
			"000-all-bindings/hooks/all",
			[]BindingType{BeforeHelm, AfterHelm, AfterDeleteHelm, OnStartup, Schedule},
			map[BindingType]float64{
				BeforeHelm:      1,
				AfterHelm:       1,
				AfterDeleteHelm: 1,
				OnStartup:       1,
			},
			[]ScheduleConfig{
				{
					Crontab:      "* * * * *",
					AllowFailure: true,
				},
			},
		},
		{
			"nested-hooks",
			"100-nested-hooks/hooks/sub/sub/nested-before-helm",
			[]BindingType{BeforeHelm},
			map[BindingType]float64{
				BeforeHelm: 1,
			},
			nil,
		},
	}

	for _, expectation := range expectations {
		t.Run(expectation.moduleName, func(t *testing.T) {
			expectedModuleHook := createModuleHook(expectation.moduleName, expectation.name, expectation.bindings, expectation.orderByBinding, expectation.schedule)

			moduleHook, err := mm.GetModuleHook(expectedModuleHook.Name)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(expectedModuleHook, moduleHook) {
				t.Errorf("\n[EXPECTED]: \n%#v\n[GOT]: \n%#v", expectedModuleHook.Hook, moduleHook.Hook)
			}
		})
	}
}

func TestMainModuleManager_GetModuleHooksInOrder2(t *testing.T) {
	mm := &MainModuleManager{}
	runInitModulesIndex(t, mm, "test_get_module_hooks_in_order")

	var expectations = []struct {
		moduleName  string
		bindingType BindingType
		hooksOrder  []string
	}{
		{
			moduleName:  "after-helm-binding-hooks",
			bindingType: AfterHelm,
			hooksOrder: []string{
				"107-after-helm-binding-hooks/hooks/b",
				"107-after-helm-binding-hooks/hooks/c",
				"107-after-helm-binding-hooks/hooks/a",
			},
		},
	}

	for _, expectation := range expectations {
		t.Run(fmt.Sprintf("%s, %s", expectation.moduleName, expectation.bindingType), func(t *testing.T) {
			moduleHooks, err := mm.GetModuleHooksInOrder(expectation.moduleName, expectation.bindingType)

			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(expectation.hooksOrder, moduleHooks) {
				t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectation.hooksOrder, moduleHooks)
			}
		})
	}
}

type MockHelmClient struct {
	helm.HelmClient
	DeleteSingleFailedRevisionExecuted bool
	UpgradeReleaseExecuted             bool
	DeleteReleaseExecuted              bool
}

func (h *MockHelmClient) CommandEnv() []string {
	return []string{}
}

func (h *MockHelmClient) DeleteSingleFailedRevision(_ string) error {
	h.DeleteSingleFailedRevisionExecuted = true
	return nil
}

func (h *MockHelmClient) UpgradeRelease(_, _ string, _ []string) error {
	h.UpgradeReleaseExecuted = true
	return nil
}

func (h *MockHelmClient) DeleteRelease(_ string) error {
	h.DeleteReleaseExecuted = true
	return nil
}

type MockKubeConfigManager struct {
	kube_config_manager.KubeConfigManager
}

func (kcm MockKubeConfigManager) SetKubeValues(values utils.Values) error {
	return nil
}

func (kcm MockKubeConfigManager) SetModuleKubeValues(moduleName string, values utils.Values) error {
	return nil
}

func TestMainModuleManager_RunModule(t *testing.T) {
	mm := &MainModuleManager{}
	hc := &MockHelmClient{}
	mm.helm = hc
	mm.kubeConfigManager = MockKubeConfigManager{}
	mm.kubeModulesConfigValues = make(map[string]utils.Values)
	runInitModulesIndex(t, mm, "test_run_module")

	moduleName := "module"
	expectedModuleDynamicValues := utils.Values{
		"afterHelm":  "override-value",
		"beforeHelm": "override-value",
	}

	err := mm.RunModule(moduleName)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedModuleDynamicValues, mm.modulesDynamicValues[moduleName]) {
		t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectedModuleDynamicValues, mm.modulesDynamicValues[moduleName])
	}

	assert.Equal(t, hc.DeleteSingleFailedRevisionExecuted, true, "helm.DeleteSingleFailedRevision must be executed!")
	assert.Equal(t, hc.UpgradeReleaseExecuted, true, "helm.UpgradeReleaseExecuted must be executed!")
}

func TestMainModuleManager_DeleteModule(t *testing.T) {
	mm := &MainModuleManager{}
	hc := &MockHelmClient{}
	mm.helm = hc
	mm.kubeConfigManager = MockKubeConfigManager{}
	mm.kubeModulesConfigValues = make(map[string]utils.Values)
	runInitModulesIndex(t, mm, "test_delete_module")

	moduleName := "module"
	expectedModuleDynamicValues := utils.Values{
		"afterDeleteHelm": "override-value",
	}

	err := mm.DeleteModule(moduleName)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedModuleDynamicValues, mm.modulesDynamicValues[moduleName]) {
		t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectedModuleDynamicValues, mm.modulesDynamicValues[moduleName])
	}

	assert.Equal(t, hc.DeleteReleaseExecuted, true, "helm.DeleteRelease must be executed!")
}

func TestMainModuleManager_RunModuleHook(t *testing.T) {
	mm := &MainModuleManager{}
	mm.helm = &MockHelmClient{}
	mm.kubeConfigManager = MockKubeConfigManager{}
	mm.kubeModulesConfigValues = make(map[string]utils.Values)
	runInitModulesIndex(t, mm, "test_run_module_hook")

	expectations := []struct {
		testName                       string
		moduleName                     string
		hookName                       string
		kubeModuleConfigValues         utils.Values
		moduleDynamicValues            utils.Values
		expectedKubeModuleConfigValues utils.Values
		expectedModuleDynamicValues    utils.Values
	}{
		{
			"merge_and_patch_kube_module_config_values",
			"update-kube-module-config",
			"000-update-kube-module-config/hooks/merge_and_patch_values",
			utils.Values{},
			utils.Values{},
			utils.Values{"a": 2.0, "c": []interface{}{3.0}},
			utils.Values{},
		},
		{
			"merge_and_patch_module_dynamic_values",
			"update-module-dynamic",
			"100-update-module-dynamic/hooks/merge_and_patch_values",
			utils.Values{},
			utils.Values{},
			utils.Values{},
			utils.Values{"a": 9.0, "c": "10"},
		},
		{
			"merge_and_patch_over_existing_kube_module_config_values",
			"update-kube-module-config",
			"000-update-kube-module-config/hooks/merge_and_patch_values",
			utils.Values{"a": 1.0, "b": 2.0, "x": "123"},
			utils.Values{},
			utils.Values{"a": 2.0, "c": []interface{}{3.0}, "x": "123"},
			utils.Values{},
		},
		{
			"merge_and_patch_over_existing_module_dynamic_values",
			"update-module-dynamic",
			"100-update-module-dynamic/hooks/merge_and_patch_values",
			utils.Values{},
			utils.Values{"a": 123.0, "x": 10.0},
			utils.Values{},
			utils.Values{"a": 9.0, "c": "10", "x": 10.0},
		},
	}

	mm.kubeModulesConfigValues = make(map[string]utils.Values)
	for _, expectation := range expectations {
		t.Run(expectation.testName, func(t *testing.T) {
			mm.kubeModulesConfigValues[expectation.moduleName] = expectation.kubeModuleConfigValues
			mm.modulesDynamicValues[expectation.moduleName] = expectation.moduleDynamicValues

			if err := mm.RunModuleHook(expectation.hookName, BeforeHelm); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(expectation.expectedKubeModuleConfigValues, mm.kubeModulesConfigValues[expectation.moduleName]) {
				t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectation.expectedKubeModuleConfigValues, mm.kubeModulesConfigValues[expectation.moduleName])
			}

			if !reflect.DeepEqual(expectation.expectedModuleDynamicValues, mm.modulesDynamicValues[expectation.moduleName]) {
				t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectation.expectedModuleDynamicValues, mm.modulesDynamicValues[expectation.moduleName])
			}
		})
	}
}

func TestMainModuleManager_GetGlobalHook2(t *testing.T) {
	mm := &MainModuleManager{}
	runInitGlobalHooks(t, mm, "test_get_global_hook")

	createGlobalHook := func(name string, bindings []BindingType, orderByBindings map[BindingType]float64, schedules []ScheduleConfig) *GlobalHook {
		globalHook := mm.newGlobalHook()
		globalHook.moduleManager = mm
		globalHook.Name = name
		globalHook.Path = filepath.Join(WorkingDir, name)
		globalHook.Schedules = schedules
		globalHook.Bindings = bindings
		globalHook.OrderByBinding = orderByBindings

		return globalHook
	}

	expectations := []struct {
		name           string
		bindings       []BindingType
		orderByBinding map[BindingType]float64
		schedule       []ScheduleConfig
	}{
		{
			"global-hooks/000-all-bindings/all",
			[]BindingType{BeforeAll, AfterAll, OnStartup, Schedule},
			map[BindingType]float64{
				BeforeAll: 1,
				AfterAll:  1,
				OnStartup: 1,
			},
			[]ScheduleConfig{
				{
					Crontab:      "* * * * *",
					AllowFailure: true,
				},
			},
		},
		{
			"global-hooks/100-nested-hook/sub/sub/nested-before-all",
			[]BindingType{BeforeAll},
			map[BindingType]float64{
				BeforeAll: 1,
			},
			nil,
		},
	}

	for _, exp := range expectations {
		t.Run(exp.name, func(t *testing.T) {
			expectedGlobalHook := createGlobalHook(exp.name, exp.bindings, exp.orderByBinding, exp.schedule)

			globalHook, err := mm.GetGlobalHook(expectedGlobalHook.Name)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(expectedGlobalHook, globalHook) {
				t.Errorf("\n[EXPECTED]: \n%#v\n[GOT]: \n%#v", expectedGlobalHook.Hook, globalHook.Hook)
			}
		})
	}
}

func TestMainModuleManager_GetGlobalHooksInOrder2(t *testing.T) {
	mm := &MainModuleManager{}
	runInitGlobalHooks(t, mm, "test_get_global_hooks_in_order")

	var expectations = []struct {
		testName    string
		bindingType BindingType
		hooksOrder  []string
	}{
		{
			testName:    "hooks",
			bindingType: AfterAll,
			hooksOrder: []string{
				"global-hooks/000-before-all-binding-hooks/b",
				"global-hooks/000-before-all-binding-hooks/c",
				"global-hooks/000-before-all-binding-hooks/a",
			},
		},
		{
			testName:    "non-supported-binding-type",
			bindingType: BeforeHelm,
			hooksOrder:  []string{},
		},
	}

	for _, expectation := range expectations {
		t.Run(expectation.testName, func(t *testing.T) {
			globalHooks := mm.GetGlobalHooksInOrder(expectation.bindingType)

			if !reflect.DeepEqual(expectation.hooksOrder, globalHooks) {
				t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectation.hooksOrder, globalHooks)
			}
		})
	}
}

func TestMainModuleManager_RunGlobalHook(t *testing.T) {
	mm := &MainModuleManager{}
	mm.helm = &MockHelmClient{}
	mm.kubeConfigManager = MockKubeConfigManager{}
	mm.kubeModulesConfigValues = make(map[string]utils.Values)
	runInitGlobalHooks(t, mm, "test_run_global_hook")

	expectations := []struct {
		testName                 string
		hookName                 string
		kubeConfigValues         utils.Values
		dynamicValues            utils.Values
		expectedKubeConfigValues utils.Values
		expectedDynamicValues    utils.Values
	}{
		{
			"merge_and_patch_kube_config_values",
			"global-hooks/000-update-kube-config/merge_and_patch_values",
			utils.Values{},
			utils.Values{},
			utils.Values{"a": 2.0, "c": []interface{}{3.0}},
			utils.Values{},
		},
		{
			"merge_and_patch_dynamic_values",
			"global-hooks/100-update-dynamic/merge_and_patch_values",
			utils.Values{},
			utils.Values{},
			utils.Values{},
			utils.Values{"a": 9.0, "c": "10"},
		},
		{
			"merge_and_patch_over_existing_kube_config_values",
			"global-hooks/000-update-kube-config/merge_and_patch_values",
			utils.Values{"a": 1.0, "b": 2.0, "x": "123"},
			utils.Values{},
			utils.Values{"a": 2.0, "c": []interface{}{3.0}, "x": "123"},
			utils.Values{},
		},
		{
			"merge_and_patch_over_existing_dynamic_values",
			"global-hooks/100-update-dynamic/merge_and_patch_values",
			utils.Values{},
			utils.Values{"a": 123.0, "x": 10.0},
			utils.Values{},
			utils.Values{"a": 9.0, "c": "10", "x": 10.0},
		},
	}

	for _, expectation := range expectations {
		t.Run(expectation.testName, func(t *testing.T) {
			mm.kubeConfigValues = expectation.kubeConfigValues
			mm.dynamicValues = expectation.dynamicValues

			if err := mm.RunGlobalHook(expectation.hookName, BeforeHelm); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(expectation.expectedKubeConfigValues, mm.kubeConfigValues) {
				t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectation.expectedKubeConfigValues, mm.kubeConfigValues)
			}

			if !reflect.DeepEqual(expectation.expectedDynamicValues, mm.dynamicValues) {
				t.Errorf("\n[EXPECTED]: %#v\n[GOT]: %#v", expectation.expectedDynamicValues, mm.dynamicValues)
			}
		})
	}
}