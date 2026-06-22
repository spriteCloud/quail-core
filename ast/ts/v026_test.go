package ts

import "testing"

func TestExtractV026_DTOInterface(t *testing.T) {
	src := []byte(`
export interface User {
  id: string;
  email: string;
  age: number;
}
`)
	syms := extractV026Symbols("src/dto.ts", src)
	if len(syms) != 1 {
		t.Fatalf("expected 1 DTO symbol; got %d (%+v)", len(syms), syms)
	}
	if syms[0].Name != "User" || !syms[0].IsDTO {
		t.Errorf("expected User+IsDTO; got %+v", syms[0])
	}
	if len(syms[0].Params) != 3 {
		t.Errorf("expected 3 fields; got %d", len(syms[0].Params))
	}
}

func TestExtractV026_RejectsInterfaceWithMethods(t *testing.T) {
	src := []byte(`
export interface Repo {
  id: string;
  save(): Promise<void>;
}
`)
	syms := extractV026Symbols("src/repo.ts", src)
	for _, s := range syms {
		if s.IsDTO {
			t.Errorf("Repo has a method — should not be a DTO: %+v", s)
		}
	}
}

func TestExtractV026_ClassWithConstructor(t *testing.T) {
	src := []byte(`
export class Order {
  constructor(id: string, total: number) {
    this.id = id;
    this.total = total;
  }
}
`)
	syms := extractV026Symbols("src/order.ts", src)
	if len(syms) != 1 {
		t.Fatalf("expected 1 class symbol; got %d", len(syms))
	}
	if syms[0].FrameworkHint != "class" || len(syms[0].Params) != 2 {
		t.Errorf("expected class+2 params; got %+v", syms[0])
	}
}

func TestExtractV026_ReduxCreateSlice(t *testing.T) {
	src := []byte(`
import { createSlice } from '@reduxjs/toolkit'
const userSlice = createSlice({
  name: 'user',
  initialState: { name: '' },
  reducers: {
    setName: (state, action) => { state.name = action.payload },
    clear: state => { state.name = '' }
  }
})
`)
	syms := extractV026Symbols("src/store/userSlice.ts", src)
	if len(syms) != 1 || syms[0].StoreKind != "redux" {
		t.Fatalf("expected one redux store; got %+v", syms)
	}
	if len(syms[0].StoreActions) < 1 {
		t.Errorf("expected ≥1 action; got %+v", syms[0].StoreActions)
	}
}

func TestExtractV026_PiniaDefineStore(t *testing.T) {
	src := []byte(`
export const useCounterStore = defineStore('counter', {
  state: () => ({ count: 0 }),
  actions: {
    increment() { this.count++ },
    reset() { this.count = 0 }
  }
})
`)
	syms := extractV026Symbols("src/stores/counter.ts", src)
	got := 0
	for _, s := range syms {
		if s.StoreKind == "pinia" {
			got++
		}
	}
	if got != 1 {
		t.Errorf("expected one pinia store; got %d (%+v)", got, syms)
	}
}

func TestExtractV026_ZustandCreate(t *testing.T) {
	src := []byte(`
export const useBearStore = create((set, get) => ({
  bears: 0,
  increase: () => set(state => ({ bears: state.bears + 1 })),
}))
`)
	syms := extractV026Symbols("src/stores/bears.ts", src)
	for _, s := range syms {
		if s.StoreKind == "zustand" {
			return
		}
	}
	t.Errorf("expected one zustand store; got %+v", syms)
}
