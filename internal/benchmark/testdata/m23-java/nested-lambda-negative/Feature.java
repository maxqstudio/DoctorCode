import java.util.function.BooleanSupplier;

class Feature {
    int classify(boolean flag) {
        BooleanSupplier supplier = () -> {
            if (flag) {
                return true;
            } else if (flag) {
                return false;
            }
            return false;
        };
        return supplier.getAsBoolean() ? 1 : 0;
    }
}
