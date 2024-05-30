// Code generated by counterfeiter. DO NOT EDIT.
package writersfakes

import (
	"sync"

	"github.com/syntasso/kratix/api/v1alpha1"
	"github.com/syntasso/kratix/lib/writers"
)

type FakeStateStoreWriter struct {
	ReadFileStub        func(string) ([]byte, error)
	readFileMutex       sync.RWMutex
	readFileArgsForCall []struct {
		arg1 string
	}
	readFileReturns struct {
		result1 []byte
		result2 error
	}
	readFileReturnsOnCall map[int]struct {
		result1 []byte
		result2 error
	}
	UpdateFilesStub        func(string, string, []v1alpha1.Workload, []string) error
	updateFilesMutex       sync.RWMutex
	updateFilesArgsForCall []struct {
		arg1 string
		arg2 string
		arg3 []v1alpha1.Workload
		arg4 []string
	}
	updateFilesReturns struct {
		result1 error
	}
	updateFilesReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeStateStoreWriter) ReadFile(arg1 string) ([]byte, error) {
	fake.readFileMutex.Lock()
	ret, specificReturn := fake.readFileReturnsOnCall[len(fake.readFileArgsForCall)]
	fake.readFileArgsForCall = append(fake.readFileArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.ReadFileStub
	fakeReturns := fake.readFileReturns
	fake.recordInvocation("ReadFile", []interface{}{arg1})
	fake.readFileMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStateStoreWriter) ReadFileCallCount() int {
	fake.readFileMutex.RLock()
	defer fake.readFileMutex.RUnlock()
	return len(fake.readFileArgsForCall)
}

func (fake *FakeStateStoreWriter) ReadFileCalls(stub func(string) ([]byte, error)) {
	fake.readFileMutex.Lock()
	defer fake.readFileMutex.Unlock()
	fake.ReadFileStub = stub
}

func (fake *FakeStateStoreWriter) ReadFileArgsForCall(i int) string {
	fake.readFileMutex.RLock()
	defer fake.readFileMutex.RUnlock()
	argsForCall := fake.readFileArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeStateStoreWriter) ReadFileReturns(result1 []byte, result2 error) {
	fake.readFileMutex.Lock()
	defer fake.readFileMutex.Unlock()
	fake.ReadFileStub = nil
	fake.readFileReturns = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *FakeStateStoreWriter) ReadFileReturnsOnCall(i int, result1 []byte, result2 error) {
	fake.readFileMutex.Lock()
	defer fake.readFileMutex.Unlock()
	fake.ReadFileStub = nil
	if fake.readFileReturnsOnCall == nil {
		fake.readFileReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 error
		})
	}
	fake.readFileReturnsOnCall[i] = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *FakeStateStoreWriter) UpdateFiles(arg1 string, arg2 string, arg3 []v1alpha1.Workload, arg4 []string) error {
	var arg3Copy []v1alpha1.Workload
	if arg3 != nil {
		arg3Copy = make([]v1alpha1.Workload, len(arg3))
		copy(arg3Copy, arg3)
	}
	var arg4Copy []string
	if arg4 != nil {
		arg4Copy = make([]string, len(arg4))
		copy(arg4Copy, arg4)
	}
	fake.updateFilesMutex.Lock()
	ret, specificReturn := fake.updateFilesReturnsOnCall[len(fake.updateFilesArgsForCall)]
	fake.updateFilesArgsForCall = append(fake.updateFilesArgsForCall, struct {
		arg1 string
		arg2 string
		arg3 []v1alpha1.Workload
		arg4 []string
	}{arg1, arg2, arg3Copy, arg4Copy})
	stub := fake.UpdateFilesStub
	fakeReturns := fake.updateFilesReturns
	fake.recordInvocation("UpdateFiles", []interface{}{arg1, arg2, arg3Copy, arg4Copy})
	fake.updateFilesMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeStateStoreWriter) UpdateFilesCallCount() int {
	fake.updateFilesMutex.RLock()
	defer fake.updateFilesMutex.RUnlock()
	return len(fake.updateFilesArgsForCall)
}

func (fake *FakeStateStoreWriter) UpdateFilesCalls(stub func(string, string, []v1alpha1.Workload, []string) error) {
	fake.updateFilesMutex.Lock()
	defer fake.updateFilesMutex.Unlock()
	fake.UpdateFilesStub = stub
}

func (fake *FakeStateStoreWriter) UpdateFilesArgsForCall(i int) (string, string, []v1alpha1.Workload, []string) {
	fake.updateFilesMutex.RLock()
	defer fake.updateFilesMutex.RUnlock()
	argsForCall := fake.updateFilesArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeStateStoreWriter) UpdateFilesReturns(result1 error) {
	fake.updateFilesMutex.Lock()
	defer fake.updateFilesMutex.Unlock()
	fake.UpdateFilesStub = nil
	fake.updateFilesReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeStateStoreWriter) UpdateFilesReturnsOnCall(i int, result1 error) {
	fake.updateFilesMutex.Lock()
	defer fake.updateFilesMutex.Unlock()
	fake.UpdateFilesStub = nil
	if fake.updateFilesReturnsOnCall == nil {
		fake.updateFilesReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.updateFilesReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeStateStoreWriter) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.readFileMutex.RLock()
	defer fake.readFileMutex.RUnlock()
	fake.updateFilesMutex.RLock()
	defer fake.updateFilesMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeStateStoreWriter) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ writers.StateStoreWriter = new(FakeStateStoreWriter)
