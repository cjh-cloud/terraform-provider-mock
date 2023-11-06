terraform {
  required_providers {
    mock = {
      source  = "tfp.cjscloud.city/cjh-cloud/mock"
      version = "0.0.1"
    }
  }
}

provider "mock" {
  foo = "test_foo_value"
}

resource "mock_example" "testing" {
  not_computed_required = "some value"

  dynamic "foo" {
    for_each = [{ number = 1 }, { number = 2 }, { number = 3 }]
    content {
      bar {
        number = foo.value.number
      }
    }
  }
  /*
   * The above is equivalent to:
   *
   * foo {
   *   bar {
   *     number = 1
   *   }
   * }
   * foo {
   *   bar {
   *     number = 2
   *   }
   * }
   * foo {
   *   bar {
   *     number = 3
   *   }
   * }
  */

  dynamic "baz" {
    // The variable inside the for_each block doesn't have to be the same as 
    // what you're assigning the value to.
    for_each = [{ something = "x" }, { something = "y" }, { something = "z" }]
    content {
      qux = baz.value.something
    }
  }
  /*
   * The above is equivalent to:
   *
   * baz {
   *   qux = "x"
   * }
   * baz {
   *   qux = "y"
   * }
   * baz {
   *   qux = "z"
   * }
  */

  some_list = ["a", "b", "c"]
}

output "last_updated" {
  value = mock_example.testing.last_updated
}
