/*
 * Zabbix Agent Bench (C) 2014  Ryan Armstrong <ryan@cavaliercoder.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"os"
	"testing"
)

func TestEnvVarExpansion(t *testing.T) {
	os.Setenv("VAR1", "Atom Eve")
	os.Setenv("VAR2", "Black Samson")
	os.Setenv("VAR3", "Cecil Stedman")
	os.Setenv("VAR4", "Darkwing")

	expected := "some.key[Atom Eve,Black Samson,,,]"
	input := "  	 some.key[{%VAR1},{%VAR2},{%Var3},{%var4},{%VAR5}]"

	key := NewItemKey(input)

	if key.Key != expected {
		t.Errorf("Environment variable subsitution failed.\nExpected: %s\nGot:      %s", expected, key.Key)
	}
}
