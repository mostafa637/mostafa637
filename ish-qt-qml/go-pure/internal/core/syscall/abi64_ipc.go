package syscall

import (
	"encoding/binary"
	"sync"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	ipcPrivate64 uint32 = 0
	ipcCreat64   uint32 = 0o1000
	ipcExcl64    uint32 = 0o2000
	ipcRmid64    uint64 = 0
	ipcSet64     uint64 = 1
	ipcStat64    uint64 = 2
	ipcInfo64    uint64 = 3

	shmRdonly64 uint64 = 0o10000
	shmRnd64    uint64 = 0o20000
	shmRemap64  uint64 = 0o40000
	shmExec64   uint64 = 0o100000

	shmStat64 uint64 = 13
	shmInfo64 uint64 = 14

	shmDSSize64       = 112
	eidrm64     int64 = -43
	enospc64    int64 = -28
)

type SharedMemorySegment64 struct {
	ID          int32
	Key         int32
	Size        uint64
	Mode        uint32
	Data        []byte
	Attachments map[corecpu.Address64]struct{}
	Marked      bool
}

type SharedMemoryRegistry64 struct {
	mu       sync.Mutex
	nextID   int32
	segments map[int32]*SharedMemorySegment64
	byKey    map[int32]int32
}

func newSharedMemoryRegistry64() *SharedMemoryRegistry64 {
	return &SharedMemoryRegistry64{nextID: 1, segments: make(map[int32]*SharedMemorySegment64), byKey: make(map[int32]int32)}
}

func sharedMemoryRegistry64(ctx *Context64) *SharedMemoryRegistry64 {
	if ctx == nil {
		return nil
	}
	if ctx.SharedMemory == nil {
		ctx.SharedMemory = newSharedMemoryRegistry64()
	}
	return ctx.SharedMemory
}

func shmget64(ctx *Context64, args [6]uint64) int64 {
	registry := sharedMemoryRegistry64(ctx)
	if registry == nil || ctx.Memory == nil {
		return int64(ENOMEM)
	}
	key := int32(args[0])
	size := args[1]
	flags := uint32(args[2])
	if size == 0 || size > uint64(^uint(0)>>1) || flags&^uint32(0o777|ipcCreat64|ipcExcl64) != 0 {
		return int64(EINVAL)
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()
	if key != int32(ipcPrivate64) {
		if id, ok := registry.byKey[key]; ok {
			segment := registry.segments[id]
			if segment == nil || segment.Marked {
				return eidrm64
			}
			if flags&ipcExcl64 != 0 && flags&ipcCreat64 != 0 {
				return int64(EEXIST)
			}
			if size > segment.Size {
				return int64(EINVAL)
			}
			return int64(id)
		}
		if flags&ipcCreat64 == 0 {
			return int64(ENOENT)
		}
	}
	id := registry.nextID
	registry.nextID++
	if id <= 0 {
		return enospc64
	}
	segment := &SharedMemorySegment64{ID: id, Key: key, Size: size, Mode: flags & 0o777, Data: make([]byte, size), Attachments: make(map[corecpu.Address64]struct{})}
	registry.segments[id] = segment
	if key != int32(ipcPrivate64) {
		registry.byKey[key] = id
	}
	return int64(id)
}

func shmat64(ctx *Context64, args [6]uint64) int64 {
	registry := sharedMemoryRegistry64(ctx)
	if registry == nil || ctx.Memory == nil {
		return int64(EINVAL)
	}
	flags := args[2]
	if flags&^(shmRdonly64|shmRnd64|shmRemap64|shmExec64) != 0 {
		return int64(EINVAL)
	}
	registry.mu.Lock()
	segment := registry.segments[int32(args[0])]
	if segment == nil || segment.Marked {
		registry.mu.Unlock()
		return eidrm64
	}
	size := segment.Size
	data := append([]byte(nil), segment.Data...)
	registry.mu.Unlock()

	addressHint := args[1]
	if flags&shmRnd64 != 0 {
		addressHint &= ^(corecpu.Page64Size - 1)
	}
	prot := uint64(ProtRead | ProtWrite)
	if flags&shmRdonly64 != 0 {
		prot = uint64(ProtRead)
	}
	mapped := mmap64(ctx, [6]uint64{addressHint, size, prot, uint64(MapShared | MapAnonymous), ^uint64(0), 0})
	if mapped < 0 {
		return mapped
	}
	base := corecpu.Address64(mapped)
	if len(data) > 0 {
		if err := ctx.Memory.Write(base, data); err != nil {
			_ = munmap64(ctx, [6]uint64{uint64(base), size})
			return int64(EFAULT)
		}
	}
	registry.mu.Lock()
	segment = registry.segments[int32(args[0])]
	if segment == nil || segment.Marked {
		registry.mu.Unlock()
		_ = munmap64(ctx, [6]uint64{uint64(base), size})
		return eidrm64
	}
	segment.Attachments[base] = struct{}{}
	registry.mu.Unlock()
	return mapped
}

func shmdt64(ctx *Context64, args [6]uint64) int64 {
	registry := sharedMemoryRegistry64(ctx)
	if registry == nil || ctx.Memory == nil || args[0]&(corecpu.Page64Size-1) != 0 {
		return int64(EINVAL)
	}
	base := corecpu.Address64(args[0])
	registry.mu.Lock()
	var segment *SharedMemorySegment64
	for _, candidate := range registry.segments {
		if _, ok := candidate.Attachments[base]; ok {
			segment = candidate
			break
		}
	}
	if segment == nil {
		registry.mu.Unlock()
		return int64(EINVAL)
	}
	data := make([]byte, len(segment.Data))
	size := segment.Size
	copy(data, segment.Data)
	delete(segment.Attachments, base)
	marked := segment.Marked && len(segment.Attachments) == 0
	registry.mu.Unlock()

	if len(data) > 0 {
		if err := ctx.Memory.Read(base, data); err != nil {
			return int64(EFAULT)
		}
	}
	if result := munmap64(ctx, [6]uint64{uint64(base), size}); result != 0 {
		return result
	}
	registry.mu.Lock()
	copy(segment.Data, data)
	if marked {
		delete(registry.segments, segment.ID)
		if registry.byKey[segment.Key] == segment.ID {
			delete(registry.byKey, segment.Key)
		}
	}
	registry.mu.Unlock()
	return 0
}

func shmctl64(ctx *Context64, args [6]uint64) int64 {
	registry := sharedMemoryRegistry64(ctx)
	if registry == nil || ctx.Memory == nil {
		return int64(EINVAL)
	}
	registry.mu.Lock()
	segment := registry.segments[int32(args[0])]
	if segment == nil {
		registry.mu.Unlock()
		return int64(EINVAL)
	}
	cmd := args[1]
	switch cmd {
	case ipcRmid64:
		segment.Marked = true
		if len(segment.Attachments) == 0 {
			delete(registry.segments, segment.ID)
			if registry.byKey[segment.Key] == segment.ID {
				delete(registry.byKey, segment.Key)
			}
		}
		registry.mu.Unlock()
		return 0
	case ipcSet64:
		if ctx.EffectiveUID != 0 {
			registry.mu.Unlock()
			return int64(EPERM)
		}
		if args[2] == 0 {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		var ds [shmDSSize64]byte
		if err := ctx.Memory.Read(corecpu.Address64(args[2]), ds[:]); err != nil {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		segment.Mode = binary.LittleEndian.Uint32(ds[24:28]) & 0o777
		registry.mu.Unlock()
		return 0
	case ipcStat64, shmStat64:
		if args[2] == 0 {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		ds := make([]byte, shmDSSize64)
		binary.LittleEndian.PutUint32(ds[0:4], uint32(segment.Key))
		binary.LittleEndian.PutUint32(ds[24:28], segment.Mode)
		binary.LittleEndian.PutUint64(ds[48:56], segment.Size)
		binary.LittleEndian.PutUint64(ds[88:96], uint64(len(segment.Attachments)))
		if err := ctx.Memory.Write(corecpu.Address64(args[2]), ds); err != nil {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		id := segment.ID
		registry.mu.Unlock()
		if cmd == shmStat64 {
			return int64(id)
		}
		return 0
	case ipcInfo64, shmInfo64:
		registry.mu.Unlock()
		return 0
	default:
		registry.mu.Unlock()
		return int64(EINVAL)
	}
}

const (
	semNowait64 uint16 = 0x1000
	semUndo64   uint16 = 0x1000
	semVMAX64   int32  = 32767

	semIPCStat64 uint64 = 2
	semIPCSet64  uint64 = 1
	semIPCRmid64 uint64 = 0
	semIPCInfo64 uint64 = 3
	semGetVal64  uint64 = 12
	semGetAll64  uint64 = 13
	semSetVal64  uint64 = 16
	semSetAll64  uint64 = 17
	semSemStat64 uint64 = 18
	semSemInfo64 uint64 = 19
	semGetPID64  uint64 = 11
	semGetNCnt64 uint64 = 14
	semGetZCnt64 uint64 = 15

	semMaxSets64 = 32000
	semMaxOps64  = 1024
	semDSSize64  = 104
)

type SemaphoreSet64 struct {
	ID      int32
	Key     int32
	Mode    uint32
	Values  []int32
	LastPID uint64
	Removed bool
	cond    *sync.Cond
}

type SemaphoreRegistry64 struct {
	mu     sync.Mutex
	nextID int32
	sets   map[int32]*SemaphoreSet64
	byKey  map[int32]int32
}

func newSemaphoreRegistry64() *SemaphoreRegistry64 {
	return &SemaphoreRegistry64{nextID: 1, sets: make(map[int32]*SemaphoreSet64), byKey: make(map[int32]int32)}
}

func semaphoreRegistry64(ctx *Context64) *SemaphoreRegistry64 {
	if ctx == nil {
		return nil
	}
	if ctx.Semaphores == nil {
		ctx.Semaphores = newSemaphoreRegistry64()
	}
	return ctx.Semaphores
}

func semget64(ctx *Context64, args [6]uint64) int64 {
	registry := semaphoreRegistry64(ctx)
	if registry == nil {
		return int64(EINVAL)
	}
	key := int32(args[0])
	nsems := args[1]
	flags := uint32(args[2])
	if nsems > semMaxSets64 || flags&^uint32(0o777|ipcCreat64|ipcExcl64) != 0 {
		return int64(EINVAL)
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if key != int32(ipcPrivate64) {
		if id, ok := registry.byKey[key]; ok {
			set := registry.sets[id]
			if set == nil || set.Removed {
				return eidrm64
			}
			if flags&ipcExcl64 != 0 && flags&ipcCreat64 != 0 {
				return int64(EEXIST)
			}
			if nsems != 0 && nsems > uint64(len(set.Values)) {
				return int64(EINVAL)
			}
			return int64(id)
		}
		if flags&ipcCreat64 == 0 {
			return int64(ENOENT)
		}
	}
	if nsems == 0 {
		return int64(EINVAL)
	}
	id := registry.nextID
	registry.nextID++
	if id <= 0 || len(registry.sets) >= semMaxSets64 {
		return enospc64
	}
	set := &SemaphoreSet64{ID: id, Key: key, Mode: flags & 0o777, Values: make([]int32, int(nsems))}
	set.cond = sync.NewCond(&registry.mu)
	registry.sets[id] = set
	if key != int32(ipcPrivate64) {
		registry.byKey[key] = id
	}
	return int64(id)
}

type semaphoreOperation64 struct {
	num  uint16
	op   int16
	flag uint16
}

func readSemaphoreOperations64(ctx *Context64, address uint64, count uint64) ([]semaphoreOperation64, int64) {
	if ctx == nil || ctx.Memory == nil || count == 0 || count > semMaxOps64 {
		return nil, int64(EINVAL)
	}
	if address > ^uint64(0)-count*6 {
		return nil, int64(EFAULT)
	}
	data := make([]byte, int(count*6))
	if err := ctx.Memory.Read(corecpu.Address64(address), data); err != nil {
		return nil, int64(EFAULT)
	}
	operations := make([]semaphoreOperation64, count)
	for index := range operations {
		offset := index * 6
		operations[index] = semaphoreOperation64{
			num:  binary.LittleEndian.Uint16(data[offset : offset+2]),
			op:   int16(binary.LittleEndian.Uint16(data[offset+2 : offset+4])),
			flag: binary.LittleEndian.Uint16(data[offset+4 : offset+6]),
		}
	}
	return operations, 0
}

func semop64(ctx *Context64, args [6]uint64) int64 {
	registry := semaphoreRegistry64(ctx)
	if registry == nil {
		return int64(EINVAL)
	}
	operations, result := readSemaphoreOperations64(ctx, args[1], args[2])
	if result != 0 {
		return result
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	set := registry.sets[int32(args[0])]
	if set == nil || set.Removed {
		return eidrm64
	}
	for {
		ready, invalid := semaphoreOperationsReady64(set, operations)
		if invalid {
			return int64(EINVAL)
		}
		if ready {
			for _, operation := range operations {
				if operation.op > 0 {
					set.Values[operation.num] += int32(operation.op)
				} else if operation.op < 0 {
					set.Values[operation.num] += int32(operation.op)
				}
			}
			set.LastPID = ctx.PID
			set.cond.Broadcast()
			return 0
		}
		if semaphoreOperationsNowait64(operations) {
			return int64(EAGAIN)
		}
		set.cond.Wait()
		if set.Removed {
			return eidrm64
		}
	}
}

func semaphoreOperationsReady64(set *SemaphoreSet64, operations []semaphoreOperation64) (ready, invalid bool) {
	for _, operation := range operations {
		if int(operation.num) >= len(set.Values) || operation.flag&^uint16(semNowait64|semUndo64) != 0 {
			return false, true
		}
		value := set.Values[operation.num]
		switch {
		case operation.op > 0:
			if value > semVMAX64-int32(operation.op) {
				return false, false
			}
		case operation.op < 0:
			if value+int32(operation.op) < 0 {
				return false, false
			}
		case value != 0:
			return false, false
		}
	}
	return true, false
}

func semaphoreOperationsNowait64(operations []semaphoreOperation64) bool {
	for _, operation := range operations {
		if operation.flag&semNowait64 != 0 {
			return true
		}
	}
	return false
}

func semctl64(ctx *Context64, args [6]uint64) int64 {
	registry := semaphoreRegistry64(ctx)
	if registry == nil || ctx == nil || ctx.Memory == nil {
		return int64(EINVAL)
	}
	registry.mu.Lock()
	set := registry.sets[int32(args[0])]
	if set == nil || set.Removed {
		registry.mu.Unlock()
		return eidrm64
	}
	cmd := args[2]
	switch cmd {
	case semIPCRmid64:
		set.Removed = true
		delete(registry.sets, set.ID)
		if registry.byKey[set.Key] == set.ID {
			delete(registry.byKey, set.Key)
		}
		set.cond.Broadcast()
		registry.mu.Unlock()
		return 0
	case semIPCSet64:
		if ctx.EffectiveUID != 0 {
			registry.mu.Unlock()
			return int64(EPERM)
		}
		if args[3] == 0 {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		var ds [semDSSize64]byte
		if err := ctx.Memory.Read(corecpu.Address64(args[3]), ds[:]); err != nil {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		set.Mode = binary.LittleEndian.Uint32(ds[24:28]) & 0o777
		registry.mu.Unlock()
		return 0
	case semIPCStat64, semSemStat64:
		if args[3] == 0 {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		ds := make([]byte, semDSSize64)
		binary.LittleEndian.PutUint32(ds[24:28], set.Mode)
		binary.LittleEndian.PutUint64(ds[64:72], uint64(len(set.Values)))
		if err := ctx.Memory.Write(corecpu.Address64(args[3]), ds); err != nil {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		id := set.ID
		registry.mu.Unlock()
		if cmd == semSemStat64 {
			return int64(id)
		}
		return 0
	case semIPCInfo64, semSemInfo64:
		registry.mu.Unlock()
		return 0
	case semGetVal64, semGetPID64, semGetNCnt64, semGetZCnt64:
		if args[1] >= uint64(len(set.Values)) && cmd == semGetVal64 {
			registry.mu.Unlock()
			return int64(EINVAL)
		}
		var value int64
		switch cmd {
		case semGetVal64:
			value = int64(set.Values[args[1]])
		case semGetPID64:
			value = int64(set.LastPID)
		default:
			value = 0
		}
		registry.mu.Unlock()
		return value
	case semSetVal64:
		if args[1] >= uint64(len(set.Values)) || args[3] > uint64(semVMAX64) {
			registry.mu.Unlock()
			return int64(EINVAL)
		}
		set.Values[args[1]] = int32(args[3])
		set.LastPID = ctx.PID
		set.cond.Broadcast()
		registry.mu.Unlock()
		return 0
	case semSetAll64:
		if args[3] == 0 {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		data := make([]byte, len(set.Values)*2)
		if err := ctx.Memory.Read(corecpu.Address64(args[3]), data); err != nil {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		for index := range set.Values {
			value := binary.LittleEndian.Uint16(data[index*2 : index*2+2])
			if value > uint16(semVMAX64) {
				registry.mu.Unlock()
				return int64(EINVAL)
			}
			set.Values[index] = int32(value)
		}
		set.LastPID = ctx.PID
		set.cond.Broadcast()
		registry.mu.Unlock()
		return 0
	case semGetAll64:
		if args[3] == 0 {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		data := make([]byte, len(set.Values)*2)
		for index, value := range set.Values {
			binary.LittleEndian.PutUint16(data[index*2:index*2+2], uint16(value))
		}
		if err := ctx.Memory.Write(corecpu.Address64(args[3]), data); err != nil {
			registry.mu.Unlock()
			return int64(EFAULT)
		}
		registry.mu.Unlock()
		return 0
	default:
		registry.mu.Unlock()
		return int64(EINVAL)
	}
}
