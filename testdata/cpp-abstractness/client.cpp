#include "shape.h"
#include "rigid.h"

void drawShape(Shape* shape) {
    shape->draw();
}

void processRigid(Rigid& r) {
    r.x = 0;
}
