import compare from "k6/x/compare";

export default function () {
  console.log(`${compare.isGreater(2, 1)}, ${compare.comparison_result}`);

  console.log(`${compare.isGreater(1, 3)}, ${compare.comparison_result}`);
}
