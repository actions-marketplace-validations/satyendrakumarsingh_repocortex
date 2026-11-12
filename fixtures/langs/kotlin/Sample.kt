package demo

import demo.util.Helper

typealias Names = List<String>

interface Shape {
    fun area(): Double
}

enum class Suit {
    HEARTS,
    SPADES
}

data class Point(val x: Int, val y: Int)

open class Circle(val r: Double) : Shape {
    override fun area(): Double = r * r * 3.14

    companion object {
        fun unit(): Circle = Circle(1.0)
    }
}

object Registry {
    fun lookup(id: Int): Circle? = null
}

fun String.shout(): String = this.uppercase()

fun main() {
    val c = Circle(2.0)
    println(c.area())
}
